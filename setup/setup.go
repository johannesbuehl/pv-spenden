package main

import (
	"database/sql"
	"fmt"
	"math/rand/v2"
	"os"
	"regexp"
	"strings"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

func createPassword(l int) string {
	passwordChars := [...]string{`A`, `B`, `C`, `D`, `E`, `F`, `G`, `H`, `I`, `J`, `K`, `L`, `M`, `N`, `O`, `P`, `Q`, `R`, `S`, `T`, `U`, `V`, `W`, `X`, `Y`, `Z`, `Ä`, `Ö`, `Ü`, `a`, `b`, `c`, `d`, `e`, `f`, `g`, `h`, `i`, `j`, `k`, `l`, `m`, `n`, `o`, `p`, `q`, `r`, `s`, `t`, `u`, `v`, `w`, `x`, `y`, `z`, `ä`, `ö`, `ü`, `ß`, `0`, `1`, `2`, `3`, `4`, `5`, `6`, `7`, `8`, `9`, `!`, `"`, `§`, `$`, `%`, `&`, `/`, `(`, `)`, `=`, `?`, `@`, `{`, `}`, `[`, `]`, `#`, `+`, `'`, `*`, `,`, `.`, `-`, `;`, `:`, `_`, `<`, `>`, `|`, `°`}
	var password string

	for ii := 0; ii < l; ii++ {
		password += passwordChars[rand.IntN(len(passwordChars))]
	}

	return password
}

func main() {
	// connect to the database
	sqlConfig := mysql.Config{
		AllowNativePasswords: true,
		Net:                  "tcp",
		User:                 config.Database.User,
		Passwd:               config.Database.Password,
		Addr:                 config.Database.Host,
		DBName:               config.Database.Database,
	}

	db, _ := sql.Open("mysql", sqlConfig.FormatDSN())

	// only proceed if the tables don't exists

	// load the sql-script
	var sqlScriptCommands []byte
	if c, err := os.ReadFile("setup.sql"); err != nil {
		panic(err)
	} else {
		sqlScriptCommands = c
	}

	// read the currently availabe tables
	if rows, err := db.Query("SHOW TABLES"); err != nil {
		panic(err)
	} else {
		defer rows.Close()

		for rows.Next() {
			var name string

			if err := rows.Scan(&name); err != nil {
				panic(err)
			} else {
				// check wether for the table there exists a create command

				if match, err := regexp.Match(fmt.Sprintf(`(?i)^create table %s`, name), sqlScriptCommands); err != nil {
					panic(err)
				} else {
					if match {
						fmt.Printf("can't setup databases: table %q already exists", name)
						os.Exit(1)
					}
				}
			}
		}
	}

	// everything is good (so far), create the tables
	for _, cmd := range strings.Split(string(sqlScriptCommands), "\n") {
		db.Exec(cmd)
	}

	// create an admin-password
	const passwordLength = 20
	password := createPassword(passwordLength)

	// hash the admin-password
	if passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost); err != nil {
		panic(err)
	} else {
		// create an admin-user
		if _, err := db.Exec("INSERT INTO users (name, password) VALUES ('admin', ?)", passwordHash); err != nil {
			panic(err)
		}
	}

	fmt.Printf(`created user "admin" with password %s\n`, password)

	// create a jwt-signature
	config.ClientSession.JwtSignature = createPassword(100)

	// write the modified config-file
	writeConfig()
}
