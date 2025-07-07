package middlewares

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func InitializeClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		print(filepath.Join(homedir.HomeDir(), ".kube", "config"))
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func SetClient(clientset *kubernetes.Clientset) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("clientset", clientset)
		c.Next()
	}
}

func GenerateToken() (string, error) {
	secret := os.Getenv("APP_AUTH_TOKEN")
	if secret == "" {
		return "", errors.New("APP_AUTH_TOKEN environment variable not set")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"namespace": "siyaha",
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			abortWithError(c, 401, "Authorization header required")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			abortWithError(c, 401, "Bearer token required")
			return
		}

		secret := os.Getenv("APP_AUTH_TOKEN")
		if secret == "" {
			abortWithError(c, 500, "APP_AUTH_TOKEN environment variable not set")
			return
		}

		token, err := parseToken(tokenString, secret)
		if err != nil {
			abortWithError(c, 401, err.Error())
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if namespace, ok := claims["namespace"].(string); ok {
				c.Set("namespace", namespace)
			} else {
				abortWithError(c, 401, "Invalid token claims")
				return
			}
		} else {
			abortWithError(c, 401, "Invalid token claims")
			return
		}

		c.Next()
	}
}

func abortWithError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{"error": message})
	c.Abort()
}

func parseToken(tokenString, secret string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
}
