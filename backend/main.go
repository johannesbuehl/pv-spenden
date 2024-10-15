package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/patrickmn/go-cache"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var db *sql.DB

var dbCache *cache.Cache

type responseMessage struct {
	Status  int
	Message *string
	Data    any
}

func dbSelect[T any](table string, where string, args ...any) ([]T, error) {
	// validate columns against struct T
	tType := reflect.TypeOf(new(T)).Elem()
	columns := make([]string, tType.NumField())

	validColumns := make(map[string]any)
	for ii := 0; ii < tType.NumField(); ii++ {
		field := tType.Field(ii)
		validColumns[strings.ToLower(field.Name)] = struct{}{}
		columns[ii] = strings.ToLower(field.Name)
	}

	for _, col := range columns {
		if _, ok := validColumns[strings.ToLower(col)]; !ok {
			return nil, fmt.Errorf("invalid column: %s for struct type %T", col, new(T))
		}
	}

	completeQuery := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), table)

	if where != "" && where != "*" {
		completeQuery = fmt.Sprintf("%s WHERE %s", completeQuery, where)
	}

	var rows *sql.Rows
	var err error

	if len(args) > 0 {
		db.Ping()

		rows, err = db.Query(completeQuery, args...)
	} else {
		db.Ping()

		rows, err = db.Query(completeQuery)
	}

	if err != nil {
		logger.Error().Msgf("database access failed with error %v", err)

		return nil, err
	}

	defer rows.Close()
	results := []T{}

	title := cases.Title(language.Und)

	for rows.Next() {
		var lineResult T

		scanArgs := make([]any, len(columns))
		v := reflect.ValueOf(&lineResult).Elem()

		for ii, col := range columns {
			colTitle := title.String(col)

			field := v.FieldByName(colTitle)

			if field.IsValid() && field.CanSet() {
				scanArgs[ii] = field.Addr().Interface()
			} else {
				logger.Warn().Msgf("Field %s not found in struct %T", col, lineResult)
				scanArgs[ii] = new(any) // save dummy value
			}
		}

		// scan the row into the struct
		if err := rows.Scan(scanArgs...); err != nil {
			logger.Warn().Msgf("Scan-error: %v", err)

			return nil, err
		}

		results = append(results, lineResult)
	}

	if err := rows.Err(); err != nil {
		logger.Error().Msgf("rows-error: %v", err)
		return nil, err
	} else {
		return results, nil
	}
}

func dbInsert(table string, vals any) error {
	// extract columns from vals
	v := reflect.ValueOf(vals)
	t := v.Type()

	columns := make([]string, t.NumField())
	values := make([]any, t.NumField())

	for ii := 0; ii < t.NumField(); ii++ {
		fieldValue := v.Field(ii)

		// skip empty (zero) values
		if !fieldValue.IsZero() {
			field := t.Field(ii)

			columns[ii] = strings.ToLower(field.Name)
			values[ii] = fieldValue.Interface()
		}
	}

	placeholders := strings.Repeat(("?, "), len(columns))
	placeholders = strings.TrimSuffix(placeholders, ", ")

	completeQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(columns, ", "), placeholders)

	_, err := db.Exec(completeQuery, values...)

	return err
}

func dbUpdate(table string, set, where any) error {
	setV := reflect.ValueOf(set)
	setT := setV.Type()

	setColumns := make([]string, setT.NumField())
	setValues := make([]any, setT.NumField())

	for ii := 0; ii < setT.NumField(); ii++ {
		fieldValue := setV.Field(ii)

		field := setT.Field(ii)

		setColumns[ii] = strings.ToLower(field.Name) + " = ?"
		setValues[ii] = fieldValue.Interface()
	}

	whereV := reflect.ValueOf(where)
	whereT := whereV.Type()

	whereColumns := make([]string, whereT.NumField())
	whereValues := make([]any, whereT.NumField())

	for ii := 0; ii < whereT.NumField(); ii++ {
		fieldValue := whereV.Field(ii)

		// skip empty (zero) values
		if !fieldValue.IsZero() {
			field := whereT.Field(ii)

			whereColumns[ii] = strings.ToLower(field.Name) + " = ?"
			whereValues[ii] = fmt.Sprint(fieldValue.Interface())
		}
	}

	sets := strings.Join(setColumns, ", ")
	wheres := strings.Join(whereColumns, " AND ")

	placeholderValues := append(setValues, whereValues...)

	completeQuery := fmt.Sprintf("UPDATE %s SET %s WHERE %s", table, sets, wheres)

	_, err := db.Exec(completeQuery, placeholderValues...)

	return err
}

func dbDelete(table string, vals any) error {
	// extract columns from vals
	v := reflect.ValueOf(vals)
	t := v.Type()

	columns := make([]string, t.NumField())
	values := make([]any, t.NumField())

	for ii := 0; ii < t.NumField(); ii++ {
		fieldValue := v.Field(ii)

		// skip empty (zero) values
		if !fieldValue.IsZero() {
			field := t.Field(ii)

			columns[ii] = strings.ToLower(field.Name) + " = ?"
			values[ii] = fmt.Sprint(fieldValue.Interface())
		}
	}

	completeQuery := fmt.Sprintf("DELETE FROM %s WHERE %s", table, strings.Join(columns, ", "))

	_, err := db.Exec(completeQuery, values...)

	return err
}

func (result responseMessage) send(c *fiber.Ctx) error {
	if result.Status >= 400 {
		if result.Message != nil {
			return fiber.NewError(result.Status, *result.Message)
		} else {
			return fiber.NewError(result.Status)
		}
	} else {
		if result.Data != nil {
			c.JSON(result.Data)
		} else {
			if result.Message != nil {
				c.SendString(*result.Message)
			}
		}

		return c.SendStatus(result.Status)
	}
}

type JWTPayload struct {
	Uid int `json:"uid"`
	Tid int `json:"tid"`
}

type JWT struct {
	Payload
	CustomClaims JWTPayload
}

func extractJWT(c *fiber.Ctx) (int, int, error) {
	cookie := c.Cookies("session")

	token, err := jwt.ParseWithClaims(cookie, &JWT{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected JWT signing method: %v", token.Header["alg"])
		}

		return []byte(config.ClientSession.JwtSignature), nil
	})

	if err != nil {
		return -1, -1, err
	}

	if claims, ok := token.Claims.(*JWT); ok && token.Valid {
		return claims.CustomClaims.Uid, claims.CustomClaims.Tid, nil
	} else {
		return -1, -1, fmt.Errorf("invalid JWT")
	}
}

func checkUser(c *fiber.Ctx) (bool, error) {
	uid, tid, err := extractJWT(c)

	if err != nil {
		return false, nil
	}

	response, err := dbSelect[UserDB]("users", "uid = ? LIMIT 1", uid)

	if err != nil {
		return false, err
	}

	return len(response) == 1 && response[0].Tid == tid, err
}

func checkAdmin(c *fiber.Ctx) (bool, error) {
	uid, tid, err := extractJWT(c)

	if err != nil {
		return false, err
	}

	response, err := dbSelect[UserDB]("users", "uid = ? LIMIT 1", uid)

	if err != nil {
		return false, err
	}

	if len(response) != 1 {
		return false, fmt.Errorf("user doesn't exist")
	} else {
		return response[0].Name == "admin" && response[0].Tid == tid, err
	}
}

type ElementDB struct {
	Mid  string `json:"mid"`
	Name string `json:"name"`
}

type ClientStatus struct {
	ReservedElements map[string]string `json:"reserved_elements"`
}

func cacheElements() error {
	if res, err := dbSelect[ElementDB]("elements", "*"); err != nil {
		return err
	} else {
		elementMap := make(map[string]string)

		for _, element := range res {
			elementMap[string(element.Mid[:])] = element.Name
		}

		dbCache.Set("elements", elementMap, cache.DefaultExpiration)

		return nil
	}
}

func getElements(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	elements, found := dbCache.Get("elements")

	if !found {
		if err := cacheElements(); err != nil {
			response.Status = fiber.StatusInternalServerError

			logger.Error().Msgf("can't get element from database: %v", err)
		} else if elements, found = dbCache.Get("elements"); !found {
			response.Status = fiber.StatusInternalServerError

			logger.Error().Msg("can't get 'elements' from cache")
		}
	}

	response.Data = ClientStatus{
		ReservedElements: elements.(map[string]string),
	}
	return response
}

var midRegex = regexp.MustCompile(`^(?:pv-\w|(?:ws|bs)-)\d{1,2}$`)

func postElements(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	if user, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError
	} else if !user {
		response.Status = fiber.StatusUnauthorized
	} else {
		body := struct{ Name string }{}

		if mid := c.Query("mid"); !midRegex.MatchString(mid) {
			response.Status = fiber.StatusBadRequest
		} else if err := c.BodyParser(&body); err != nil {
			logger.Warn().Msg(`body can't be parsed as "struct{ name string }"`)

			response.Status = fiber.StatusBadRequest
		} else {
			// clear the current cache
			dbCache.Delete("elements")

			// write the data to the database
			if err := dbInsert("elements", ElementDB{Mid: mid, Name: body.Name}); err != nil {
				response.Status = fiber.StatusInternalServerError
			} else {
				response = getElements(c)
			}
		}
	}

	return response
}

func patchElements(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	if user, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError
	} else if !user {
		response.Status = fiber.StatusUnauthorized
	} else {
		body := struct{ Name string }{}

		if mid := c.Query("mid"); !midRegex.MatchString(mid) {
			response.Status = fiber.StatusBadRequest
		} else if err := c.BodyParser(&body); err != nil {
			logger.Warn().Msg(`body can't be parsed as "struct{ name string }"`)

			response.Status = fiber.StatusBadRequest
		} else {
			// clear the current cache
			dbCache.Delete("elements")

			// write the data to the database
			if err := dbUpdate("elements", struct{ Name string }{Name: body.Name}, struct{ Mid string }{Mid: mid}); err != nil {
				response.Status = fiber.StatusInternalServerError
			} else {
				response = getElements(c)
			}
		}
	}

	return response
}

func deleteElements(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	if user, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError
	} else if !user {
		response.Status = fiber.StatusUnauthorized
	} else {
		if mid := c.Query("mid"); !midRegex.MatchString(mid) {
			response.Status = fiber.StatusBadRequest
		} else {
			dbCache.Delete("elements")

			if err := dbDelete("elements", struct{ Mid string }{Mid: mid}); err != nil {
				response.Status = fiber.StatusInternalServerError
			} else {
				response = getElements(c)
			}
		}
	}

	return response
}

type AddUser struct {
	Name     string
	Password string
}

type UserDB struct {
	Uid      int    `json:"uid"`
	Name     string `json:"name"`
	Password []byte `json:"password"`
	Tid      int    `json:"tid"`
}

func hashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

func getUsers(c *fiber.Ctx) responseMessage {
	var response responseMessage

	if isAdmin, err := checkAdmin(c); err != nil {
		response.Status = fiber.StatusInternalServerError
	} else if !isAdmin {
		response.Status = fiber.StatusUnauthorized
	} else {
		// retrieve all users
		if users, err := dbSelect[struct {
			Uid  int    `json:"uid"`
			Name string `json:"name"`
		}]("users", ""); err != nil {
			response.Status = fiber.StatusInternalServerError

			logger.Error().Msgf("can't get users from database: %v", err)
		} else {
			response.Data = users
		}
	}

	return response
}

func validatePassword(password string) bool {
	return len(password) >= 12 && len(password) <= 64
}

func postUsers(c *fiber.Ctx) responseMessage {
	response := responseMessage{}
	body := AddUser{}

	if admin, err := checkAdmin(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("error while checking for admin-user: %v", err)
	} else if !admin {
		response.Status = fiber.StatusUnauthorized
	} else if err := c.BodyParser(&body); err != nil {
		logger.Warn().Msg(`body can't be parsed as "struct{ name string; Password string }"`)

		response.Status = fiber.StatusBadRequest
	} else {
		// check, wether the name already exists
		if dbUsers, err := dbSelect[UserDB]("users", "name = ? LIMIT 1", body.Name); err != nil {
			response.Status = fiber.StatusInternalServerError
		} else if len(dbUsers) != 0 {
			response.Status = fiber.StatusBadRequest

			logger.Info().Msgf("can't add user: user with name %q already exists", body.Name)
		} else {
			// everything is valid
			if hashedPassword, err := hashPassword(body.Password); err != nil {
				logger.Error().Msgf("error during password-hashing: %v", err)

				response.Status = fiber.StatusInternalServerError
			} else {
				if err := dbInsert("users", struct {
					Name     string
					Password []byte
				}{Name: body.Name, Password: hashedPassword}); err != nil {
					logger.Error().Msgf("can't add user to database: %v", err)

					response.Status = fiber.StatusInternalServerError
				} else {
					response = getUsers(c)
				}
			}
		}
	}

	return response
}

func patchUsers(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	if admin, err := checkAdmin(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Info().Msgf("error while check for admin: %v", err)
	} else if !admin {
		response.Status = fiber.StatusUnauthorized

		logger.Info().Msg("user is no admin")
	} else {
		body := struct {
			Password string `json:"password"`
		}{}

		// check wether a valid uid is present
		if uid := c.QueryInt("uid", -1); uid < 0 {
			logger.Info().Msg("query doesn't include valid uid")

			response.Status = fiber.StatusBadRequest
		} else {
			// try to parse the body
			if err := c.BodyParser(&body); err != nil {
				logger.Warn().Msg(`body can't be parsed as "struct{ password string }"`)

				response.Status = fiber.StatusBadRequest
			} else {
				// check, wether the user exists
				if dbUsers, err := dbSelect[UserDB]("users", "uid = ? LIMIT 1", uid); err != nil {
					response.Status = fiber.StatusInternalServerError
				} else if len(dbUsers) != 1 {
					response.Status = fiber.StatusBadRequest

					logger.Info().Msgf("can't modify user: user with uid %q doesn't exist", uid)
				} else {
					// everything is valid

					// hash the new password
					if hashedPassword, err := hashPassword(body.Password); err != nil {
						logger.Error().Msgf("error during password-hashing: %v", err)

						response.Status = fiber.StatusInternalServerError
					} else {
						// increase the token-id of the user to make the current-token invalid
						if _, err := incTokenId(uid); err != nil {
							response.Status = fiber.StatusInternalServerError
						} else {
							// update the databse with the new password
							if err := dbUpdate("users", struct{ Password []byte }{Password: hashedPassword}, struct{ Uid int }{Uid: uid}); err != nil {
								logger.Error().Msgf("can't update password: %v", err)

								response.Status = fiber.StatusInternalServerError
							} else {
								return getUsers(c)
							}
						}
					}
				}
			}
		}
	}

	return response
}

func deleteUsers(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	if admin, err := checkAdmin(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("error while checking for admin-user: %v", err)
	} else if !admin {
		response.Status = fiber.StatusUnauthorized

		// check wether there is a valid uid
	} else if uid := c.QueryInt("uid", -1); uid < 0 {
		logger.Info().Msg("query doesn't include valid uid")

		response.Status = fiber.StatusBadRequest
	} else {
		// delete the user from the database
		if err := dbDelete("users", struct{ Uid int }{Uid: uid}); err != nil {
			response.Status = fiber.StatusInternalServerError

			logger.Error().Msgf("can't delete user with uid = %q: %v", uid, err)
		} else {
			response = getUsers(c)
		}
	}

	return response
}

func patchUserPassword(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	if user, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Info().Msgf("error during user-check: %v", err)
	} else if !user {
		response.Status = fiber.StatusUnauthorized
	} else {
		// parse the body
		var body struct {
			Password string `json:"password"`
		}

		if uid, _, err := extractJWT(c); err != nil {
			response.Status = fiber.StatusBadRequest

			logger.Info().Msg("can't extract uid from query")
		} else if err := c.BodyParser(&body); err != nil {
			response.Status = fiber.StatusBadRequest

			logger.Warn().Msg(`body can't be parsed as "struct{ password string }"`)
		} else if !validatePassword(body.Password) {
			response.Status = fiber.StatusBadRequest

			response.Message = ptr("invalid password")

			logger.Warn().Msg("invalid password")
		} else {
			// hash the new password
			if hashedPassword, err := hashPassword(body.Password); err != nil {
				logger.Error().Msgf("error during password-hashing: %v", err)

				response.Status = fiber.StatusInternalServerError
			} else {
				// increase the token-id of the user to make the current-token invalid
				if _, err := incTokenId(uid); err != nil {
					response.Status = fiber.StatusInternalServerError
				} else {
					// update the databse with the new password
					if err := dbUpdate("users", struct{ Password []byte }{Password: hashedPassword}, struct{ Uid int }{Uid: uid}); err != nil {
						logger.Error().Msgf("can't update password: %v", err)

						response.Status = fiber.StatusInternalServerError
					} else {
						return getUsers(c)
					}
				}
			}
		}
	}

	return response
}

func handleWelcome(c *fiber.Ctx) error {
	response := responseMessage{}
	response.Data = UserLogin{
		LoggedIn: false,
	}

	if ok, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError
	} else if !ok {
		response.Status = fiber.StatusUnauthorized
	} else {
		if uid, _, err := extractJWT(c); err != nil {
			response.Status = fiber.StatusBadRequest
		} else {
			if users, err := dbSelect[UserDB]("users", "uid = ? LIMIT 1", strconv.Itoa(uid)); err != nil {
				response.Status = fiber.StatusInternalServerError
			} else {
				if len(users) != 1 {
					response.Status = fiber.StatusForbidden
					response.Message = ptr("unknown user")

					removeSessionCookie(c)
				} else {
					user := users[0]

					response.Data = UserLogin{
						Uid:      user.Uid,
						Name:     user.Name,
						LoggedIn: true,
					}
				}
			}
		}
	}

	return response.send(c)
}

type LoginBody struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type UserLogin struct {
	Uid      int    `json:"uid"`
	Name     string `json:"name"`
	LoggedIn bool   `json:"logged_in"`
}

func getTokenId(uid int) (int, error) {
	if response, err := dbSelect[UserDB]("users", "uid = ? LIMIT 1", uid); err != nil {
		return -1, err
	} else if len(response) != 1 {
		return -1, fmt.Errorf("can't get user with uid = %q from database", uid)
	} else {
		return response[0].Tid, nil
	}
}

func incTokenId(uid int) (int, error) {
	if tid, err := getTokenId(uid); err != nil {
		return -1, err
	} else {
		if err := dbUpdate("users", struct{ Tid int }{Tid: tid}, struct{ Uid int }{Uid: uid}); err != nil {
			return -1, err
		} else {
			return tid, nil
		}
	}
}

func handleLogin(c *fiber.Ctx) error {
	var response responseMessage

	body := LoginBody{}

	if err := c.BodyParser(&body); err != nil {
		logger.Warn().Msgf("error while parsing login-body: %v", err)

		response.Status = fiber.StatusBadRequest
	} else {
		// try to get the hashed password from the database
		dbResult, err := dbSelect[UserDB]("users", "name = ? LIMIT 1", body.User)

		if err != nil {
			response.Status = fiber.StatusInternalServerError
		} else if len(dbResult) != 1 {
			response.Status = fiber.StatusForbidden
		} else {
			response.Data = UserLogin{
				LoggedIn: false,
			}

			user := dbResult[0]

			if len(dbResult) != 1 || bcrypt.CompareHashAndPassword(user.Password, []byte(body.Password)) != nil {
				response.Status = fiber.StatusUnauthorized
				message := "Unkown user or wrong password"
				response.Message = &message
			} else {
				// get the token-id
				if tid, err := incTokenId(user.Uid); err != nil {
					response.Status = fiber.StatusInternalServerError

					logger.Error().Msgf("can't get a new tid for user with uid = %q", user.Uid)
				} else {
					// create the jwt
					jwt, err := config.signJWT(JWTPayload{
						Uid: user.Uid,
						Tid: tid,
					})

					if err != nil {
						logger.Error().Msgf("json-webtoken creation failed: %v", err)

						response.Status = fiber.StatusInternalServerError
					} else {

						c.Cookie(&fiber.Cookie{
							Name:     "session",
							Value:    jwt,
							HTTPOnly: true,
							SameSite: "strict",
							MaxAge:   int(config.SessionExpire.Seconds()),
						})

						response.Data = UserLogin{
							Uid:      user.Uid,
							Name:     user.Name,
							LoggedIn: true,
						}
					}
				}
			}
		}
	}

	return response.send(c)
}

func removeSessionCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "session",
		Value:    "",
		HTTPOnly: true,
		SameSite: "strict",
		Expires:  time.Unix(0, 0),
	})
}

func handleLogout(c *fiber.Ctx) error {
	removeSessionCookie(c)

	return responseMessage{
		Data: UserLogin{
			LoggedIn: false,
		},
	}.send(c)
}

func main() {
	// database
	sqlConfig := mysql.Config{
		AllowNativePasswords: true,
		Net:                  "tcp",
		User:                 config.Database.User,
		Passwd:               config.Database.Password,
		Addr:                 config.Database.Host,
		DBName:               config.Database.Database,
	}

	db, _ = sql.Open("mysql", sqlConfig.FormatDSN())
	db.SetMaxIdleConns(10)
	db.SetMaxIdleConns(100)
	db.SetConnMaxLifetime(time.Minute)

	// cache
	dbCache = cache.New(config.Cache.Expiration, config.Cache.Purge)

	app := fiber.New(fiber.Config{
		AppName:               "johannes-pv",
		DisableStartupMessage: true,
	})

	handleMethods := map[string]func(path string, handlers ...func(*fiber.Ctx) error) fiber.Router{
		"GET":    app.Get,
		"POST":   app.Post,
		"PATCH":  app.Patch,
		"DELETE": app.Delete,
	}

	endpoints := map[string]map[string]func(*fiber.Ctx) responseMessage{
		"GET": {
			"elements": getElements,
			"users":    getUsers,
		},
		"POST": {
			"elements": postElements,
			"users":    postUsers,
		},
		"PATCH": {
			"elements":      patchElements,
			"users":         patchUsers,
			"user/password": patchUserPassword,
		},
		"DELETE": {
			"elements": deleteElements,
			"users":    deleteUsers,
		},
	}

	// handle specific requests special
	app.Get("/pv/api/welcome", handleWelcome)
	app.Post("/pv/api/login", handleLogin)
	app.Get("/pv/api/logout", handleLogout)

	for method, handlers := range endpoints {
		for address, handler := range handlers {
			handleMethods[method]("/pv/api/"+address, func(c *fiber.Ctx) error {
				logger.Debug().Msgf("HTTP %s request: %q", c.Method(), c.OriginalURL())

				return handler(c).send(c)
			})
		}
	}

	app.Listen(fmt.Sprintf(":%d", config.Server.Port))
}
