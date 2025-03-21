package webpush

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt"
)

func TestVAPID(t *testing.T) {
	for _, subIsURL := range []bool{false, true} {
		t.Run(fmt.Sprintf("subIsUrl=%v", subIsURL), func(t *testing.T) {
			s := getStandardEncodedTestSubscription()
			var sub string
			if subIsURL {
				sub = "https://test.com/"
			} else {
				sub = "test@test.com"
			}

			// Generate vapid keys
			vapidPrivateKey, vapidPublicKey, err := GenerateVAPIDKeys()
			if err != nil {
				t.Fatal(err)
			}

			// Get authentication header
			vapidAuthHeader, err := getVAPIDAuthorizationHeader(
				s.Endpoint,
				sub,
				subIsURL,
				vapidPublicKey,
				vapidPrivateKey,
			)
			if err != nil {
				t.Fatal(err)
			}

			// Validate the token in the Authorization header
			tokenString := getTokenFromAuthorizationHeader(vapidAuthHeader, t)

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
					t.Fatal("Wrong validation method need ECDSA!")
				}

				// To decode the token it needs the VAPID public key
				b64 := base64.RawURLEncoding
				decodedVapidPrivateKey, err := b64.DecodeString(vapidPrivateKey)
				if err != nil {
					t.Fatal("Could not decode VAPID private key")
				}

				privKey := generateVAPIDHeaderKeys(decodedVapidPrivateKey)
				return privKey.Public(), nil
			})

			// Check the claims on the token
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				var expectedSub string
				if subIsURL {
					expectedSub = sub
				} else {
					expectedSub = fmt.Sprintf("mailto:%s", sub)
				}
				if expectedSub != claims["sub"] {
					t.Fatalf(
						"Incorreect mailto, expected=%s, got=%s",
						expectedSub,
						claims["sub"],
					)
				}

				if claims["aud"] == "" {
					t.Fatal("Audience should not be empty")
				}
			} else {
				t.Fatal(err)
			}
		})
	}
}

func TestDecodeVapidKeyAcceptsURLEncoding(t *testing.T) {
	_, err := decodeVapidKey("8Xq5BzDqeP9TqEfjt6ZB4TygtMd1ogwstfbAM3KF1W4=")
	if err != nil {
		t.Fatal(err)
	}
}

func TestDecodeVapidKeyAcceptsRawURLEncoding(t *testing.T) {
	_, err := decodeVapidKey("MHcCAQEEIHF7ijDrb8gwj_9o7UuSx9t_oGlPMyOsG9YQLp3qJwLuoAoGCCqGSM49AwEHoUQDQgAEhB-nJdg0d5oOkdTYsKqbbuQ06ZUYkS0H-ELXsShIkpmcIVIO16Sj15YMBouesMbY4xPdepwF4Pj3QfaALRAG5Q")
	if err != nil {
		t.Fatal(err)
	}
}

func TestVAPIDKeys(t *testing.T) {
	privateKey, publicKey, err := GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}

	if len(privateKey) != 43 {
		t.Fatal("Generated incorrect VAPID private key")
	}

	if len(publicKey) != 87 {
		t.Fatal("Generated incorrect VAPID public key")
	}
}

// Helper function for extracting the token from the Authorization header
func getTokenFromAuthorizationHeader(tokenHeader string, t *testing.T) string {
	hsplit := strings.Split(tokenHeader, " ")
	if len(hsplit) < 3 {
		t.Fatal("Failed to auth split header")
	}

	tsplit := strings.Split(hsplit[1], "=")
	if len(tsplit) < 2 {
		t.Fatal("Failed to t split header on =")
	}

	return tsplit[1][:len(tsplit[1])-1]
}
