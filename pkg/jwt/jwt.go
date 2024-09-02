package jwt

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/nifetency/nife.io/utils"
)

// secret key being used to sign tokens
var (
	SecretKey = []byte("secret")
)

// Claims is data we save in each token
type Claims struct {
	email string
	jwt.StandardClaims
}

//GenerateAccessToken generates a jwt token and assign a email to it's claims and return it
func GenerateAccessToken(email string, productId string, isResetPassword bool, firstName string, lastName string, companyName string, roleId int, customerStripeId string) (string, error) {

	accessTokenExpiryTime := utils.GetEnv("ACCESS_TOKEN_EXIPRY_TIME", "60")
	if isResetPassword {
		accessTokenExpiryTime = utils.GetEnv("PASSWORD_RESET_TOKEN_EXPIRY", "60")
	}
	tokenString, err := GenerateToken(email, productId, accessTokenExpiryTime, firstName, lastName, companyName, roleId, customerStripeId)
	if err != nil {
		log.Fatal("Error in Generating Access Token")
		return "", err
	}
	return tokenString, nil
}

//ParseToken parses a jwt token and returns the username it it's claims
func ParseToken(tokenStr string) (string, string, string, string, string, int, string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return SecretKey, nil
	})
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		var Email string
		if value, ok := claims["email"].(string); ok {
			Email = value
		} else if value, ok := claims["ssoExternalId"].(string); ok {
			Email = value
		}
		stripesProductId := claims["productId"].(string)
		firstName := claims["firstName"].(string)
		lastName := claims["lastName"].(string)
		companyName := claims["companyName"].(string)
		roleId := claims["roleId"].(float64)
		customerStripeId := claims["customerStripeId"].(string)
		return Email, stripesProductId, firstName, lastName, companyName, int(roleId), customerStripeId, nil
	} else {
		return "", "", "", "", "", 0, "", err
	}
}

//GenerateRefreshToken generates a jwt token and assign a email to it's claims and return it
func GenerateRefreshToken(email string, productId string, firstName string, lastName string, companyName string, roleId int, customerStripeId string) (string, error) {
	refreshTokenExpiryTime := utils.GetEnv("REFRESH_TOKEN_EXIPRY_TIME", "1440")
	tokenString, err := GenerateToken(email, productId, refreshTokenExpiryTime, firstName, lastName, companyName, roleId, customerStripeId)
	if err != nil {
		log.Fatal("Error in Generating Refresh Token")
		return "", err
	}
	return tokenString, nil
}

// GenerateToken generates the jwt token for given email
func GenerateToken(email string, productId string, tokenExpiryTime string, firstName string, lastName string, companyName string, roleId int, customerStripeId string) (string, error) {
	tokenExpiryTimeInt := utils.StringToInt64(tokenExpiryTime)
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	Email := IsEmailValid(email)
	if Email {
		claims["email"] = email
	} else {
		claims["ssoExternalId"] = email
	}
	claims["productId"] = productId
	claims["firstName"] = firstName
	claims["lastName"] = lastName
	claims["companyName"] = companyName
	claims["roleId"] = roleId
	claims["customerStripeId"] = customerStripeId
	claims["exp"] = time.Now().Add(time.Minute * time.Duration(tokenExpiryTimeInt)).Unix()
	tokenString, err := token.SignedString(SecretKey)
	return tokenString, err
}

func IsEmailValid(e string) bool {
	emailRegex := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return emailRegex.MatchString(e)
}

func ValidateJWTTokenFormat(accessToken string) error {

	_, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// Define the secret key used for signing
		secret := []byte("secret")
		return secret, nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "signature is invalid") {
			return fmt.Errorf("Token signature is invalid")
		} else {
			return fmt.Errorf("Token signature is invalid")
		}
	}

	// // Validate the token claims
	// if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
	// 	fmt.Println("Token claims:", claims)
	// } else {
	// 	fmt.Println("Token is invalid")
	// }
	return nil
}
