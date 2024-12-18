package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/patrickmn/go-cache"
	mail "github.com/xhit/go-simple-mail/v2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// connection to database
var db *sql.DB

// cache for database
var dbCache *cache.Cache

// general message for REST-responses
type responseMessage struct {
	Status  int
	Message string
	Data    any
}

// query the database
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

	// create the query
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

// insert data intot the databse
func dbInsert(table string, vals any) error {
	// extract columns from vals
	v := reflect.ValueOf(vals)
	t := v.Type()

	columns := make([]string, t.NumField())
	values := make([]any, t.NumField())

	for ii := 0; ii < t.NumField(); ii++ {
		fieldValue := v.Field(ii)

		field := t.Field(ii)

		columns[ii] = strings.ToLower(field.Name)
		values[ii] = fieldValue.Interface()
	}

	placeholders := strings.Repeat(("?, "), len(columns))
	placeholders = strings.TrimSuffix(placeholders, ", ")

	completeQuery := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(columns, ", "), placeholders)

	_, err := db.Exec(completeQuery, values...)

	return err
}

// update data in the database
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

// remove data from the database
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

// answer the client request with the response-message
func (result responseMessage) send(c *fiber.Ctx) error {
	// if the status-code is in the error-region, return an error
	if result.Status >= 400 {
		// if available, include the message
		if result.Message != "" {
			return fiber.NewError(result.Status, result.Message)
		} else {
			return fiber.NewError(result.Status)
		}
	} else {
		// if there is data, send it as JSON
		if result.Data != nil {
			c.JSON(result.Data)

			// if there is a message, send it instead
		} else if result.Message != "" {
			c.SendString(result.Message)
		}

		return c.SendStatus(result.Status)
	}
}

// payload of the JSON webtoken
type JWTPayload struct {
	Uid int `json:"uid"`
	Tid int `json:"tid"`
}

// complete JSON webtoken
type JWT struct {
	Payload
	CustomClaims JWTPayload
}

// extracts the json webtoken from the request
//
// @returns (uID, tID, error)
func extractJWT(c *fiber.Ctx) (int, int, error) {
	// get the session-cookie
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

	// extract the claims from the JWT
	if claims, ok := token.Claims.(*JWT); ok && token.Valid {
		return claims.CustomClaims.Uid, claims.CustomClaims.Tid, nil
	} else {
		return -1, -1, fmt.Errorf("invalid JWT")
	}
}

func setSessionCookie(c *fiber.Ctx, jwt *string) {
	var value string

	if jwt == nil {
		value = c.Cookies("session")
	} else {
		value = *jwt
	}

	c.Cookie(&fiber.Cookie{
		Name:     "session",
		Value:    value,
		HTTPOnly: true,
		SameSite: "strict",
		MaxAge:   int(config.SessionExpire.Seconds()),
	})
}

// checks wether the request is from a valid user
func checkUser(c *fiber.Ctx) (bool, error) {
	uid, tid, err := extractJWT(c)

	if err != nil {
		return false, nil
	}

	// retrieve the user from the database
	response, err := dbSelect[UserDB]("users", "uid = ? LIMIT 1", uid)

	if err != nil {
		return false, err
	}

	// if exactly one user came back and the tID is valid, the user is authorized
	if len(response) == 1 && response[0].Tid == tid {
		// reset the expiration of the cookie
		setSessionCookie(c, nil)

		return true, err
	} else {
		return false, err
	}
}

// checks wether the request is from the admin
func checkAdmin(c *fiber.Ctx) (bool, error) {
	uid, tid, err := extractJWT(c)

	if err != nil {
		return false, err
	}

	// retrieve the user from the database
	response, err := dbSelect[UserDB]("users", "uid = ? LIMIT 1", uid)

	if err != nil {
		return false, err
	}

	// if exactly one user came back and its name is "admin", the user is the admin
	if len(response) != 1 {
		return false, fmt.Errorf("user doesn't exist")
	} else {
		return response[0].Name == "admin" && response[0].Tid == tid, err
	}
}

// information about an element in the database
type ElementDB struct {
	Mid         string  `json:"mid"`
	Name        string  `json:"name"`
	Reservation *string `json:"reservation"`
	Mail        *string `json:"mail"`
}

type ElementDBNoReservation struct {
	Mid  string  `json:"mid"`
	Name string  `json:"name"`
	Mail *string `json:"mail"`
}

// client-data of the reserved elements
type ClientStatus struct {
	Taken    map[string]string `json:"taken"`
	Reserved []string          `json:"reserved"`
}

type ElementsCache struct {
	Taken    map[string]string
	Reserved []string
}

// caches the elements from the database
func cacheElements() error {
	if res, err := dbSelect[ElementDB]("elements", "*"); err != nil {
		return err
	} else {
		// delete all expired reservations
		var expiredElements []any
		expirationDate := time.Now().Add(-config.Reservation.Expiration)

		takenElements := make(map[string]string)
		reservedElements := []string{}

		for _, element := range res {
			if element.Reservation != nil {
				if reservationDate, err := time.Parse(time.DateTime, *element.Reservation); err == nil {
					if reservationDate.Sub(expirationDate) < 0 {
						expiredElements = append(expiredElements, element.Mid)

						continue
					}
				}

				reservedElements = append(reservedElements, element.Mid)
			} else {
				takenElements[element.Mid] = element.Name
			}
		}

		if len(expiredElements) > 0 {
			// remove the expired elements from the database
			if _, err := db.Exec(fmt.Sprintf("DELETE FROM elements WHERE mid IN (%s?)", strings.Repeat("?, ", len(expiredElements)-1)), expiredElements...); err != nil {
				logger.Error().Msgf("can't remove expired elements from database: %v", err)

				return err
			}
		}

		dbCache.Set("elements", ElementsCache{
			Taken:    takenElements,
			Reserved: reservedElements,
		}, cache.DefaultExpiration)

		return nil
	}
}

// gets the elements from the cache
func getElements(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	elements, found := dbCache.Get("elements")

	if !found {
		if err := cacheElements(); err != nil {
			response.Status = fiber.StatusInternalServerError
			response.Message = "can't get elements"

			logger.Error().Msgf("can't get elements from database: %v", err)
		} else if elements, found = dbCache.Get("elements"); !found {
			response.Status = fiber.StatusInternalServerError
			response.Message = "can't get elements"

			logger.Error().Msg(`can't get "elements" from cache`)
		}
	}

	// if the reponse-status is still unset, there was no error
	if response.Status == 0 {

		response.Data = ClientStatus{
			Taken:    elements.(ElementsCache).Taken,
			Reserved: elements.(ElementsCache).Reserved,
		}

		logger.Debug().Msg("retrieved elements")
	}

	return response
}

// regex to match valid element-names
func isValidMid(element string) (bool, error) {
	if results := config.MidRegex.FindStringSubmatch(element); results == nil {
		return false, nil
	} else {
		// check wether the descriptor-part is valid
		if rng, ok := config.ValidateElements.ValidElements[results[1]]; !ok {
			return false, nil

			// try to parse the mid-number
		} else if n, err := strconv.Atoi(results[2]); err != nil {
			return false, err
		} else {
			return rng.From <= n && n <= rng.To, nil
		}
	}
}

// handles post-requests for reserving new elements
func postElements(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	body := struct {
		Name string
		Mail string
	}{}

	mid := c.Query("mid")

	if ok, err := isValidMid(mid); err != nil || !ok {
		response.Status = fiber.StatusBadRequest
		response.Message = "invalid mID"

		logger.Info().Msgf("can't reserve element: invalid element-name: %q", mid)
	} else if err := c.BodyParser(&body); err != nil {
		response.Status = fiber.StatusBadRequest
		response.Message = "invalid message-body"

		logger.Warn().Msg(`body can't be parsed as "struct{ name string mail string}"`)
	} else {
		elements, found := dbCache.Get("elements")

		if !found {
			if err := cacheElements(); err != nil {
				response.Status = fiber.StatusInternalServerError
				response.Message = "can't get elements"

				logger.Error().Msgf("can't get elements from database: %v", err)
			} else if elements, found = dbCache.Get("elements"); !found {
				response.Status = fiber.StatusInternalServerError
				response.Message = "can't get elements"

				logger.Error().Msg("can't get 'elements' from cache")
			}
		}

		// if the status is still unset, there was no error
		if response.Status == 0 {
			// check wether the element already exists
			if _, ok := elements.(ElementsCache).Taken[mid]; ok {
				response.Status = fiber.StatusBadRequest
				response.Message = "element is already taken"

				logger.Info().Msgf("element %q is already taken", mid)

				return response
			} else if slices.Contains(elements.(ElementsCache).Reserved, mid); ok {
				response.Status = fiber.StatusBadRequest
				response.Message = "element is currently reserved"

				logger.Info().Msgf("element %q is currently reserved", mid)
			}

			// send the reservation e-mail
			data := ReservationData{
				Mail: body.Mail,
				Mid:  mid,
				Name: body.Name,
			}

			if err := data.sendReservationEmail(); err != nil {
				logger.Error().Msgf("can't send reservation-mail: %v", err)
			} else {
				// clear the current cache
				dbCache.Delete("elements")

				// write the data to the database
				if err := dbInsert("elements", ElementDBNoReservation{Mid: mid, Name: body.Name, Mail: &body.Mail}); err != nil {
					response.Status = fiber.StatusInternalServerError
					response.Message = "error while writing reservation to database"

					logger.Error().Msgf("can't write reservation to database: %v", err)
				} else {
					response = getElements(c)

					logger.Debug().Msgf("reserved element %q", mid)
				}
			}
		}
	}

	return response
}

func getElementType(mid string) string {
	switch strings.Split(mid, "-")[0] {
	case "pv":
		return "PV-Modul"
	case "bs":
		return "Batteriespeicher"
	default:
		return ""
	}
}

func getElementArticle(mid string) string {
	switch strings.Split(mid, "-")[0] {
	case "pv":
		return "das"
	case "bs":
		return "den"
	default:
		return ""
	}
}

func getElementID(mid string) string {
	return strings.ToUpper(strings.Split(mid, "-")[1])
}

type ReservationData struct {
	Mail string
	Mid  string
	Name string
}

func (data ReservationData) sendReservationEmail() error {
	email := mail.NewMSG()

	templateData := SponsorshipTemplateData{}
	templateData.populate(data.Mid, data.Name)

	if subject, err := parseTemplate("templates/reservation_mail", templateData); err != nil {
		return err
	} else if bodyHTML, err := parseHTMLTemplate("templates/reservation_mail.html", templateData); err != nil {
		return err
	} else if bodyPlain, err := parseHTMLTemplate("templates/reservation_mail.txt", templateData); err != nil {
		return err
	} else {
		email.SetFrom(fmt.Sprintf("Klimaplus-Patenschaft <%s>", config.Mail.User)).AddTo(data.Mail).SetSubject(subject)

		email.SetBody(mail.TextPlain, bodyPlain)

		email.AddAlternative(mail.TextHTML, bodyHTML)

		if mailClient, err := mailServer.Connect(); err != nil {
			logger.Fatal().Msgf("can't connect to to mail-server: %v", err)

			return err
		} else if err := email.Send(mailClient); err != nil {
			return err
		} else {
			return nil
		}
	}
}

// handles patch-requests for modifying element reservations
func patchElements(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	if user, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check user: %v", err)
	} else if !user {
		response.Status = fiber.StatusUnauthorized

		logger.Info().Msg("request is not authorized as user")
	} else {
		body := struct{ Name string }{}

		mid := c.Query("mid")
		if ok, err := isValidMid(mid); err != nil || !ok {
			response.Status = fiber.StatusBadRequest
			response.Message = "invalid element name"

			logger.Info().Msgf("can't modify element: invalid element-name: %q", mid)
		} else if err := c.BodyParser(&body); err != nil {
			response.Status = fiber.StatusBadRequest
			response.Message = "invalid message-body"

			logger.Warn().Msg(`body can't be parsed as "struct{ name string }"`)
		} else {
			// check wether the element already exists
			if elements, found := dbCache.Get("elements"); found {
				if _, ok := elements.(map[string]string)[mid]; !ok {
					response.Status = fiber.StatusBadRequest
					response.Message = "element is already reserved"

					logger.Info().Msgf("element %q is already reserved", mid)

					return response
				}
			}

			// clear the current cache
			dbCache.Delete("elements")

			// write the data to the database
			if err := dbUpdate("elements", struct{ Name string }{Name: body.Name}, struct{ Mid string }{Mid: mid}); err != nil {
				response.Status = fiber.StatusInternalServerError
				response.Message = "error while writing reservation to database"

				logger.Error().Msgf("can't write reservation to database: %v", err)
			} else {
				response = getElements(c)

				logger.Debug().Msgf("modified reservation for element %q", mid)
			}
		}
	}

	return response
}

// handle delete-requets for deleting an element reservation
func deleteElements(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	if user, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check user: %v", err)
	} else if !user {
		response.Status = fiber.StatusUnauthorized

		logger.Info().Msg("request is not authorized as user")
	} else {
		mid := c.Query("mid")

		if ok, err := isValidMid(mid); !ok || err != nil {
			response.Status = fiber.StatusBadRequest
			response.Message = "invalid element name"

			logger.Info().Msgf("can't delete element: invalid element-name: %q", mid)
		} else {
			dbCache.Delete("elements")

			if err := dbDelete("elements", struct{ Mid string }{Mid: mid}); err != nil {
				response.Status = fiber.StatusInternalServerError
				response.Message = "error while deleting reservation from database"

				logger.Error().Msgf("can't delete reservation from database: %v", err)
			} else {
				response = getElements(c)

				logger.Debug().Msgf("deleted reservation for %q", mid)
			}
		}
	}

	return response
}

// request for adding a user
type AddUserBody struct {
	Name     string
	Password string
}

// user-entry in the database
type UserDB struct {
	Uid      int    `json:"uid"`
	Name     string `json:"name"`
	Password []byte `json:"password"`
	Tid      int    `json:"tid"`
}

// hashes a password
func hashPassword(password string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
}

// handles get-request for the users
func getUsers(c *fiber.Ctx) responseMessage {
	var response responseMessage

	if isAdmin, err := checkAdmin(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check for admin-user: %v", err)
	} else if !isAdmin {
		response.Status = fiber.StatusUnauthorized

		logger.Info().Msg("request is not authorized as admin")
	} else {
		// retrieve all users
		if users, err := dbSelect[struct {
			Uid  int    `json:"uid"`
			Name string `json:"name"`
		}]("users", ""); err != nil {
			response.Status = fiber.StatusInternalServerError
			response.Message = "can't get users from database"

			logger.Error().Msgf("can't get users from database: %v", err)
		} else {
			response.Data = users

			logger.Debug().Msg("retrieved users from database")
		}
	}

	return response
}

func getReservations(c *fiber.Ctx) responseMessage {
	var response responseMessage

	if ok, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check for user: %v", err)
	} else if !ok {
		response.Status = fiber.StatusUnauthorized

		logger.Info().Msg("request in not authorized")
	} else if res, err := dbSelect[ElementDB]("elements", "reservation IS NOT NULL"); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't get reserved elements from database: %v", err)
	} else {

		response.Data = res
	}

	return response
}

func getSponsorships(c *fiber.Ctx) responseMessage {
	var response responseMessage

	if ok, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check for user: %v", err)
	} else if !ok {
		response.Status = fiber.StatusUnauthorized

		logger.Info().Msg("request in not authorized")
	} else {
		if res, err := dbSelect[ElementDBNoReservation]("elements", "reservation IS NULL"); err != nil {
			response.Status = fiber.StatusInternalServerError

			logger.Error().Msgf("can't get sponsored elements from database: %v", err)
		} else {

			response.Data = res
		}
	}

	return response
}

func getCertificates(c *fiber.Ctx) responseMessage {
	var response responseMessage

	if ok, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check for user: %v", err)
	} else if !ok {
		response.Status = fiber.StatusUnauthorized
	} else if mid := c.Query("mid"); mid == "" {
		response.Status = fiber.StatusBadRequest
		response.Message = "query doesn't include mid"

		logger.Info().Msg("query doesn't include mid")
	} else {
		// get the element from the database
		if res, err := dbSelect[ElementDB]("elements", "mid = ?", mid); err != nil {
			response.Status = fiber.StatusInternalServerError

			logger.Error().Msgf("can't get element %q from database: %v", mid, err)
		} else if len(res) != 1 {
			response.Status = fiber.StatusBadRequest
			response.Message = "query doesn't include valid mid"

			logger.Info().Msgf("query doesn't include valid mid: %q", mid)
		} else {
			// create the pdf
			certData := CertificateData{
				Reservation: ReservationData{
					Mid:  mid,
					Name: res[0].Name,
				},
			}

			if err := certData.create(); err != nil {
				response.Status = fiber.StatusInternalServerError

				logger.Error().Msgf("can't create certificate for %q; %v", mid, err)
			} else {
				defer certData.cleanup()

				c.Attachment(certData.PDFFile)
				c.SendFile(certData.PDFFile)
			}
		}
	}

	return response
}

// validates a password against the password-rules
func validatePassword(password string) bool {
	return len(password) >= 12 && len(password) <= 64
}

// handles post-request to add a new user to the database
func postUsers(c *fiber.Ctx) responseMessage {
	response := responseMessage{}
	body := AddUserBody{}

	if admin, err := checkAdmin(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check for admin-user: %v", err)
	} else if !admin {
		response.Status = fiber.StatusUnauthorized

		logger.Info().Msg("request is not authorized as user")
	} else if err := c.BodyParser(&body); err != nil {
		response.Status = fiber.StatusBadRequest
		response.Message = "invalid message-body"

		logger.Warn().Msg(`body can't be parsed as "struct{ name string; Password string }"`)
	} else {
		if dbUsers, err := dbSelect[UserDB]("users", "name = ? LIMIT 1", body.Name); err != nil {
			response.Status = fiber.StatusInternalServerError

			logger.Error().Msgf("can't read users from database: %v", err)
		} else if len(dbUsers) != 0 {
			response.Status = fiber.StatusBadRequest
			response.Message = "user already exists"

			logger.Info().Msgf("can't add user: user with name %q already exists", body.Name)
		} else {
			// everything is valid
			if hashedPassword, err := hashPassword(body.Password); err != nil {
				response.Status = fiber.StatusInternalServerError

				logger.Error().Msgf("can't hash password: %v", err)
			} else {
				if err := dbInsert("users", struct {
					Name     string
					Password []byte
				}{Name: body.Name, Password: hashedPassword}); err != nil {
					response.Status = fiber.StatusInternalServerError
					response.Message = "can't add user to database"

					logger.Error().Msgf("can't add user to database: %v", err)
				} else {
					response = getUsers(c)

					logger.Debug().Msgf("added user %q", body.Name)
				}
			}
		}
	}

	return response
}

func postReservations(c *fiber.Ctx) responseMessage {
	var response responseMessage

	if ok, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check for user: %v", err)
	} else if !ok {
		response.Status = fiber.StatusUnauthorized

		// check if mid is in query
	} else if mid := c.Query("mid"); mid == "" {
		response.Status = fiber.StatusBadRequest
		response.Message = "query doesn't include valid mid"

		logger.Info().Msg("query doesn't include valid mid")
	} else if userData, err := dbSelect[ElementDB]("elements", "mid = ?", mid); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't retrieve element-data for %q: %v", mid, err)
	} else if len(userData) != 1 {
		response.Status = fiber.StatusNotFound
		response.Message = "no reservation found"

		logger.Info().Msgf("no element-reservation for %q", mid)
	} else {
		// create the certificate and send it via e-mail
		certData := CertificateData{
			Reservation: ReservationData{
				Mid:  mid,
				Name: userData[0].Name,
				Mail: *userData[0].Mail,
			},
		}

		defer certData.cleanup()

		if err := certData.create(); err != nil {
			response.Status = fiber.StatusInternalServerError
			response.Message = "error while creating certificate"

			logger.Error().Msgf("can't create certificate for %q: %v", mid, err)
		} else if err := certData.send(); err != nil {
			response.Status = fiber.StatusInternalServerError
			response.Message = "error while sending certificate"

			logger.Error().Msgf("can't send certificate for %q: %v", mid, err)
		} else if err := dbUpdate("elements", struct {
			Reservation *string
			Mail        *string
		}{}, struct{ Mid string }{Mid: mid}); err != nil {
			response.Status = fiber.StatusInternalServerError

			logger.Error().Msgf("can't write reservation-confirm to database for %q: %v", mid, err)
		} else {
			dbCache.Delete("elements")
		}

		response = getReservations(c)
	}

	return response
}

// change the password in the database
func changePassword(uid int, password string) responseMessage {
	response := responseMessage{}

	// hash the new password
	if hashedPassword, err := hashPassword(password); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't hash password: %v", err)
	} else {
		// increase the token-id of the user to make the current-token invalid
		if err := incTokenId(uid); err != nil {
			response.Status = fiber.StatusInternalServerError

			logger.Error().Msgf("can't increase the tid: %v", err)
		} else {
			// update the databse with the new password
			if err := dbUpdate("users", struct{ Password []byte }{Password: hashedPassword}, struct{ Uid int }{Uid: uid}); err != nil {
				response.Status = fiber.StatusInternalServerError
				response.Message = "can't update password"

				logger.Error().Msgf("can't update password: %v", err)
			} else {
				logger.Debug().Msgf("updated password for user %q", uid)

				response.Status = fiber.StatusOK
			}
		}
	}

	return response
}

// handles patch-request to change a useres password
func patchUsers(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	if admin, err := checkAdmin(c); err != nil {
		response.Status = fiber.StatusInternalServerError
		response.Message = "error while checking the authorization"

		logger.Error().Msgf("can't check for admin: %v", err)
	} else if !admin {
		response.Status = fiber.StatusUnauthorized

		logger.Info().Msg("user is no admin")
	} else {
		body := struct {
			Password string `json:"password"`
		}{}

		// check wether a valid uid is present
		if uid := c.QueryInt("uid", -1); uid < 0 {
			response.Status = fiber.StatusBadRequest
			response.Message = "query doesn't include valid uid"

			logger.Info().Msg("query doesn't include valid uid")
		} else {
			// try to parse the body
			if err := c.BodyParser(&body); err != nil {
				response.Status = fiber.StatusBadRequest
				response.Message = "invalid message-body"

				logger.Warn().Msg(`body can't be parsed as "struct{ password string }"`)
			} else {
				// check, wether the user exists
				if dbUsers, err := dbSelect[UserDB]("users", "uid = ? LIMIT 1", uid); err != nil {
					response.Status = fiber.StatusInternalServerError

					logger.Error().Msgf("can't read users from database: %v", err)
				} else if len(dbUsers) != 1 {
					response.Status = fiber.StatusBadRequest
					response.Message = "user doesn't exist"

					logger.Info().Msgf("can't modify user: user with uid %q doesn't exist", uid)
				} else {
					// everything is valid

					if response = changePassword(uid, body.Password); response.Status == fiber.StatusOK {
						response = getUsers(c)
					}
				}
			}
		}
	}

	return response
}

// handle delete-request for removing a user
func deleteUsers(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	if admin, err := checkAdmin(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check for admin-user: %v", err)
	} else if !admin {
		response.Status = fiber.StatusUnauthorized

		// check wether there is a valid uid
	} else if uid := c.QueryInt("uid", -1); uid < 0 {
		response.Status = fiber.StatusBadRequest
		response.Message = "query doesn't include valid uid"

		logger.Info().Msg("query doesn't include valid uid")
	} else {
		// delete the user from the database
		if err := dbDelete("users", struct{ Uid int }{Uid: uid}); err != nil {
			response.Status = fiber.StatusInternalServerError
			response.Message = "can't delete user"

			logger.Error().Msgf("can't delete user with uid = %q: %v", uid, err)
		} else {
			logger.Debug().Msgf("deleted user with uid = %q", uid)

			response = getUsers(c)
		}
	}

	return response
}

func deleteReservations(c *fiber.Ctx) responseMessage {
	var response responseMessage

	if ok, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check for user: %v", err)
	} else if !ok {
		response.Status = fiber.StatusUnauthorized

		// check for mid in query
	} else if mid := c.Query("mid"); mid == "" {
		response.Status = fiber.StatusBadRequest
		response.Message = "query doesn't include valid mid"

		logger.Info().Msg("query doesn't include valid mid")
	} else {
		if err := dbDelete("elements", struct{ Mid string }{Mid: mid}); err != nil {
			response.Status = fiber.StatusInternalServerError

			logger.Error().Msgf("error while removing reservation for element %q from database: %v", mid, err)
		} else {
			dbCache.Delete("elements")

			response = getReservations(c)
		}
	}

	return response
}

func deleteSponsorships(c *fiber.Ctx) responseMessage {
	var response responseMessage

	if ok, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check for user: %v", err)
	} else if !ok {
		response.Status = fiber.StatusUnauthorized

		// check for mid in query
	} else if mid := c.Query("mid"); mid == "" {
		response.Status = fiber.StatusBadRequest
		response.Message = "query doesn't include valid mid"

		logger.Info().Msg("query doesn't include valid mid")
	} else {
		if err := dbDelete("elements", struct{ Mid string }{Mid: mid}); err != nil {
			response.Status = fiber.StatusInternalServerError

			logger.Error().Msgf("error while removing sponsorship for element %q from database: %v", mid, err)
		} else {
			dbCache.Delete("elements")

			response = getSponsorships(c)
		}
	}

	return response
}

// handles patch-requests to change the users password
func patchUserPassword(c *fiber.Ctx) responseMessage {
	response := responseMessage{}

	if user, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check for user: %v", err)
	} else if !user {
		response.Status = fiber.StatusUnauthorized

		logger.Info().Msg("request is not authorized as user")
	} else {
		// parse the body
		var body struct {
			Password string `json:"password"`
		}

		if uid, _, err := extractJWT(c); err != nil {
			response.Status = fiber.StatusBadRequest
			response.Message = "query doesn't include valid uid"

			logger.Warn().Msg("can't extract uid from query")
		} else if err := c.BodyParser(&body); err != nil {
			response.Status = fiber.StatusBadRequest

			logger.Warn().Msg(`body can't be parsed as "struct{ password string }"`)
		} else if !validatePassword(body.Password) {
			response.Status = fiber.StatusBadRequest
			response.Message = "invalid password"

			logger.Info().Msg("invalid password")
		} else {
			// everything is valid

			return changePassword(uid, body.Password)
		}
	}

	return response
}

func patchReservations(c *fiber.Ctx) responseMessage {
	var response responseMessage

	if ok, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check user: %v", err)
	} else if !ok {
		response.Status = fiber.StatusUnauthorized

		// check for mid in query
	} else if mid := c.Query("mid"); mid == "" {
		response.Status = fiber.StatusBadRequest
		response.Message = "query doesn't include valid mid"

		logger.Info().Msg("query doesn't include valid mid")
	} else {
		// parse the body
		body := struct{ Name string }{}

		if err := c.BodyParser(&body); err != nil {
			response.Status = fiber.StatusBadRequest

			logger.Warn().Msg(`body can't be parsed as "struct{ name string }"`)
		} else {
			// update the database with the new name
			dbUpdate("elements", body, struct{ Mid string }{Mid: mid})

			dbCache.Delete("elements")

			response = getReservations(c)
		}
	}

	return response
}

func patchSponsorships(c *fiber.Ctx) responseMessage {
	var response responseMessage

	if ok, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Error().Msgf("can't check user: %v", err)
	} else if !ok {
		response.Status = fiber.StatusUnauthorized

		// check for mid in query
	} else if mid := c.Query("mid"); mid == "" {
		response.Status = fiber.StatusBadRequest
		response.Message = "query doesn't include valid mid"

		logger.Info().Msg("query doesn't include valid mid")
	} else {
		// parse the body
		body := struct{ Name string }{}

		if err := c.BodyParser(&body); err != nil {
			response.Status = fiber.StatusBadRequest

			logger.Warn().Msg(`body can't be parsed as "struct{ name string }"`)
		} else {
			// update the database with the new name
			dbUpdate("elements", body, struct{ Mid string }{Mid: mid})

			dbCache.Delete("elements")

			response = getSponsorships(c)
		}
	}

	return response
}

// handle welcome-messages from clients
func handleWelcome(c *fiber.Ctx) error {
	logger.Debug().Msgf("HTTP %s request: %q", c.Method(), c.OriginalURL())

	response := responseMessage{}
	response.Data = UserLogin{
		LoggedIn: false,
	}

	if ok, err := checkUser(c); err != nil {
		response.Status = fiber.StatusInternalServerError

		logger.Warn().Msgf("can't check user: %v", err)
	} else if !ok {
		response.Status = fiber.StatusNoContent
	} else {
		if uid, _, err := extractJWT(c); err != nil {
			response.Status = fiber.StatusBadRequest

			logger.Error().Msgf("can't extract JWT: %v", err)
		} else {
			if users, err := dbSelect[UserDB]("users", "uid = ? LIMIT 1", strconv.Itoa(uid)); err != nil {
				response.Status = fiber.StatusInternalServerError

				logger.Error().Msgf("can't get users from database: %v", err)
			} else {
				if len(users) != 1 {
					response.Status = fiber.StatusForbidden
					response.Message = "unknown user"

					removeSessionCookie(c)
				} else {
					user := users[0]

					response.Data = UserLogin{
						Uid:      user.Uid,
						Name:     user.Name,
						LoggedIn: true,
					}
				}

				logger.Debug().Msgf("welcomed user with uid = %v", uid)
			}
		}
	}

	return response.send(c)
}

// body from a login-request
type LoginBody struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// data of the logged-in-status
type UserLogin struct {
	Uid      int    `json:"uid"`
	Name     string `json:"name"`
	LoggedIn bool   `json:"logged_in"`
}

// retrieves the current tid for a specific user from the database
func getTokenId(uid int) (int, error) {
	if response, err := dbSelect[UserDB]("users", "uid = ? LIMIT 1", uid); err != nil {
		return -1, err
	} else if len(response) != 1 {
		return -1, fmt.Errorf("can't get user with uid = %q from database", uid)
	} else {
		return response[0].Tid, nil
	}
}

// increases the tid of a user
func incTokenId(uid int) error {
	_, err := db.Exec("UPDATE users SET tid = tid + 1 WHERE uid = ?", uid)

	return err
}

var messageWrongLogin = "Unkown user or wrong password"

// handles login-requests
func handleLogin(c *fiber.Ctx) error {
	logger.Debug().Msgf("HTTP %s request: %q", c.Method(), c.OriginalURL())

	var response responseMessage

	body := LoginBody{}

	if err := c.BodyParser(&body); err != nil {
		response.Status = fiber.StatusBadRequest
		response.Message = "can't parse message-body"

		logger.Warn().Msgf("can't parse login-body: %v", err)
	} else {
		// try to get the hashed password from the database
		dbResult, err := dbSelect[UserDB]("users", "name = ? LIMIT 1", body.User)

		if err != nil {
			response.Status = fiber.StatusInternalServerError

			logger.Error().Msgf("can't get users from the database: %v", err)
		} else if len(dbResult) != 1 {
			response.Status = fiber.StatusForbidden
			response.Message = messageWrongLogin

			logger.Info().Msgf("user with name = %q doesn't exist", body.User)
		} else {
			response.Data = UserLogin{
				LoggedIn: false,
			}

			user := dbResult[0]

			if len(dbResult) != 1 || bcrypt.CompareHashAndPassword(user.Password, []byte(body.Password)) != nil {
				response.Status = fiber.StatusUnauthorized
				response.Message = messageWrongLogin

				logger.Debug().Msgf("can't login: wrong username or password")
			} else {
				// get the token-id
				if tid, err := getTokenId(user.Uid); err != nil {
					response.Status = fiber.StatusInternalServerError

					logger.Error().Msgf("can't get tid for user with uid = %q", user.Uid)
				} else {
					// create the jwt
					jwt, err := config.signJWT(JWTPayload{
						Uid: user.Uid,
						Tid: tid,
					})

					if err != nil {
						response.Status = fiber.StatusInternalServerError

						logger.Error().Msgf("json-webtoken creation failed: %v", err)
					} else {
						setSessionCookie(c, &jwt)

						response.Data = UserLogin{
							Uid:      user.Uid,
							Name:     user.Name,
							LoggedIn: true,
						}

						logger.Info().Msgf("user with uid = %q logged in", user.Uid)
					}
				}
			}
		}
	}

	return response.send(c)
}

// removes the session-coockie from a request
func removeSessionCookie(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "session",
		Value:    "",
		HTTPOnly: true,
		SameSite: "strict",
		Expires:  time.Unix(0, 0),
	})
}

// handles logout-requests
func handleLogout(c *fiber.Ctx) error {
	logger.Debug().Msgf("HTTP %s request: %q", c.Method(), c.OriginalURL())

	removeSessionCookie(c)

	return responseMessage{
		Data: UserLogin{
			LoggedIn: false,
		},
	}.send(c)
}

func main() {
	// setup the database-connection
	sqlConfig := mysql.Config{
		AllowNativePasswords: true,
		Net:                  "tcp",
		User:                 config.Database.User,
		Passwd:               config.Database.Password,
		Addr:                 config.Database.Host,
		DBName:               config.Database.Database,
	}

	// connect to the database
	db, _ = sql.Open("mysql", sqlConfig.FormatDSN())
	db.SetMaxIdleConns(10)
	db.SetMaxIdleConns(100)
	db.SetConnMaxLifetime(time.Minute)

	// setup the cache
	dbCache = cache.New(config.Cache.Expiration, config.Cache.Purge)

	// setup fiber
	app := fiber.New(fiber.Config{
		AppName:               "johannes-pv",
		DisableStartupMessage: true,
	})

	// map with the individual methods
	handleMethods := map[string]func(path string, handlers ...func(*fiber.Ctx) error) fiber.Router{
		"GET":    app.Get,
		"POST":   app.Post,
		"PATCH":  app.Patch,
		"DELETE": app.Delete,
	}

	// map with the individual registered endpoints
	endpoints := map[string]map[string]func(*fiber.Ctx) responseMessage{
		"GET": {
			"elements":     getElements,
			"users":        getUsers,
			"reservations": getReservations,
			"sponsorships": getSponsorships,
			"certificates": getCertificates,
		},
		"POST": {
			"elements":     postElements,
			"users":        postUsers,
			"reservations": postReservations,
		},
		"PATCH": {
			"elements":      patchElements,
			"users":         patchUsers,
			"user/password": patchUserPassword,
			"reservations":  patchReservations,
			"sponsorships":  patchSponsorships,
		},
		"DELETE": {
			"elements":     deleteElements,
			"users":        deleteUsers,
			"reservations": deleteReservations,
			"sponsorships": deleteSponsorships,
		},
	}

	// handle specific requests special
	app.Get("/api/welcome", handleWelcome)
	app.Post("/api/login", handleLogin)
	app.Get("/api/logout", handleLogout)

	// register the registered endpoints
	for method, handlers := range endpoints {
		for address, handler := range handlers {
			handleMethods[method]("/api/"+address, func(c *fiber.Ctx) error {
				logger.Debug().Msgf("HTTP %s request: %q", c.Method(), c.OriginalURL())

				return handler(c).send(c)
			})
		}
	}

	// start the server
	app.Listen(fmt.Sprintf(":%d", config.Server.Port))
}
