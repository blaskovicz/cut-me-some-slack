package chat

import (
	"crypto/subtle"
	"fmt"

	randomdata "github.com/Pallinder/go-randomdata"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/mitchellh/mapstructure"
)

const TokenVersion = "1"
const TokenISS = "cut-me-some-slack"

func generateSignedJWT(hmacKey []byte) (*User, string, error) {
	user := User{Username: fmt.Sprintf("anonymous-%s-%d", randomdata.Noun(), randomdata.Number(5000))}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":  TokenISS,
		"sub":  user.Username,
		"user": user,
		"tv":   TokenVersion,
		//"exp": TODO
		//"nbf": time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
	})

	tokenString, err := token.SignedString(hmacKey)
	return &user, tokenString, err
}

func verifySignedJWT(hmacKey []byte, tokenString string) (*User, *jwt.Token, error) {
	var user *User
	// TODO maybe just use jwt.ParseWithClaims()
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("Malformed claims received: %v", token.Claims)
		}

		// validate issuer
		if !claims.VerifyIssuer(TokenISS, true) {
			return nil, fmt.Errorf("Unexpected token issuer: %v", claims["iss"])
		}

		// validate token version
		tvRaw, _ := claims["tv"].(string)
		if subtle.ConstantTimeCompare([]byte(tvRaw), []byte(TokenVersion)) != 1 {
			return nil, fmt.Errorf("Unexpected token version: %s", tvRaw)
		}

		// extract/validate the user
		var u User
		err := mapstructure.Decode(claims["user"], &u)
		if err != nil || u.Username == "" {
			return nil, fmt.Errorf("Invalid user claim: %v", claims["user"])
		}
		user = &u

		// TODO exp
		return hmacKey, nil
	})
	return user, token, err
}
