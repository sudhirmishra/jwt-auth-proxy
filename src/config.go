package main

import (
	"log"
	"math/rand"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	JwtSigningKey           string
	PublicListenAddr        string
	PublicAPIPath           string
	BackendListenAddr       string
	BackendCertDir          string
	BackendCertHostnames    []string
	BackendCertIPs          []net.IP
	BackendGenerateCert     bool
	TemplateSignup          string
	TemplateChangeEmail     string
	TemplateResetPassword   string
	TemplateNewPassword     string
	MongoDbURL              string
	MongoDbName             string
	EnableCors              bool
	CorsOrigin              string
	CorsHeaders             string
	SMTPServer              string
	SMTPSenderAddr          string
	AllowSignup             bool
	AllowChangePassword     bool
	AllowChangeEmail        bool
	AllowForgotPassword     bool
	AllowDeleteAccount      bool
	EnableTOTP              bool
	TOTPIssuer              string
	TOTPSecretEncryptionKey string
	ProxyTarget             *url.URL
	ProxyWhitelist          []string
	ProxyBlacklist          []string
	AccessTokenLifetime     time.Duration
	RefreshTokenLifetime    time.Duration
	PendingActionLifetime   time.Duration
}

var _configInstance *Config
var _configOnce sync.Once

func GetConfig() *Config {
	_configOnce.Do(func() {
		_configInstance = &Config{}
		_configInstance.ReadConfig()
	})
	return _configInstance
}

func (c *Config) GenerateRandomPassword(length int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"123456789")
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}

func (c *Config) ReadConfig() {
	log.Println("Reading config...")
	c.JwtSigningKey = c._GetEnv("JWT_SIGNING_KEY", c.GenerateRandomPassword(32))
	c.PublicListenAddr = c._GetEnv("PUBLIC_LISTEN_ADDR", "0.0.0.0:8080")
	c.PublicAPIPath = c._GetEnv("PUBLIC_API_PATH", "/auth/")
	if !strings.HasSuffix(c.PublicAPIPath, "/") {
		c.PublicAPIPath += "/"
	}
	c.BackendListenAddr = c._GetEnv("BACKEND_LISTEN_ADDR", "0.0.0.0:8443")
	c.BackendCertDir = c._GetEnv("BACKEND_CERT_DIR", "./certs/")
	if !strings.HasSuffix(c.BackendCertDir, "/") {
		c.BackendCertDir += "/"
	}
	c.BackendCertHostnames = strings.Split(c._GetEnv("BACKEND_CERT_HOSTNAMES", "localhost"), ",")
	splitIPs := strings.Split(c._GetEnv("BACKEND_CERT_IPS", "127.0.0.1,::1"), ",")
	c.BackendCertIPs = make([]net.IP, len(splitIPs))
	for i, ipAddr := range splitIPs {
		c.BackendCertIPs[i] = net.ParseIP(ipAddr)
	}
	c.BackendGenerateCert = (c._GetEnv("BACKEND_GENERATE_CERT", "1") == "1")
	c.TemplateSignup = c._GetEnv("TEMPLATE_SIGNUP", "res/signup.tpl")
	c.TemplateChangeEmail = c._GetEnv("TEMPLATE_CHANGE_EMAIL", "res/changeemail.tpl")
	c.TemplateResetPassword = c._GetEnv("TEMPLATE_RESET_PASSWORD", "res/resetpassword.tpl")
	c.TemplateNewPassword = c._GetEnv("TEMPLATE_NEW_PASSWORD", "res/newpassword.tpl")
	c.MongoDbURL = c._GetEnv("MONGO_DB_URL", "mongodb://localhost:27017")
	c.MongoDbName = c._GetEnv("MONGO_DB_NAME", "jwt_auth_proxy")
	c.EnableCors = (c._GetEnv("CORS_ENABLE", "0") == "1")
	c.CorsOrigin = c._GetEnv("CORS_ORIGIN", "*")
	c.CorsHeaders = c._GetEnv("CORS_HEADERS", "*")
	c.SMTPServer = c._GetEnv("SMTP_SERVER", "127.0.0.1:25")
	c.SMTPSenderAddr = c._GetEnv("SMTP_SENDER_ADDR", "no-reply@localhost")
	c.AllowSignup = (c._GetEnv("ALLOW_SIGNUP", "1") == "1")
	c.AllowChangePassword = (c._GetEnv("ALLOW_CHANGE_PASSWORD", "1") == "1")
	c.AllowChangeEmail = (c._GetEnv("ALLOW_CHANGE_EMAIL", "1") == "1")
	c.AllowForgotPassword = (c._GetEnv("ALLOW_FORGOT_PASSWORD", "1") == "1")
	c.AllowDeleteAccount = (c._GetEnv("ALLOW_DELETE_ACCOUNT", "1") == "1")
	c.EnableTOTP = (c._GetEnv("TOTP_ENABLE", "0") == "1")
	c.TOTPIssuer = c._GetEnv("TOTP_ISSUER", "JWT Auth Proxy")
	c.TOTPSecretEncryptionKey = c._GetEnv("TOTP_ENCRYPT_KEY", "")
	if c.EnableTOTP && len(c.TOTPSecretEncryptionKey) < 16 {
		log.Fatal("TOTP_ENCRYPT_KEY with minimum length of 16 bytes required")
	}
	if proxyTaget, err := url.Parse(c._GetEnv("PROXY_TARGET", "http://127.0.0.1:80")); err != nil {
		log.Fatal(err)
	} else {
		c.ProxyTarget = proxyTaget
	}
	c.ProxyWhitelist = strings.Split(strings.TrimSpace(c._GetEnv("PROXY_WHITELIST", "")), ":")
	if len(c.ProxyWhitelist) == 1 && c.ProxyWhitelist[0] == "" {
		c.ProxyWhitelist = make([]string, 0)
	}
	c.ProxyBlacklist = strings.Split(strings.TrimSpace(c._GetEnv("PROXY_BLACKLIST", "")), ":")
	if len(c.ProxyBlacklist) == 1 && c.ProxyBlacklist[0] == "" {
		c.ProxyBlacklist = make([]string, 0)
	}
	if len(c.ProxyBlacklist) > 0 && len(c.ProxyWhitelist) > 0 {
		log.Fatal("Can't set both PROXY_WHITELIST and PROXY_BLACKLIST")
	}
	if i, err := strconv.Atoi(c._GetEnv("ACCESS_TOKEN_LIFETIME", "5")); err != nil {
		log.Fatal(err)
	} else {
		c.AccessTokenLifetime = time.Duration(i)
	}
	if i, err := strconv.Atoi(c._GetEnv("REFRESH_TOKEN_LIFETIME", strconv.Itoa(24*60))); err != nil {
		log.Fatal(err)
	} else {
		c.RefreshTokenLifetime = time.Duration(i)
	}
	if i, err := strconv.Atoi(c._GetEnv("PENDING_ACTION_LIFETIME", strconv.Itoa(24*60))); err != nil {
		log.Fatal(err)
	} else {
		c.PendingActionLifetime = time.Duration(i)
	}
}

func (c *Config) _GetEnv(key, defaultValue string) string {
	res := os.Getenv(key)
	if res == "" {
		return defaultValue
	}
	return res
}
