package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jyogi-oauth/auth-server/internal/config"
	"github.com/jyogi-oauth/auth-server/internal/handler"
	mw "github.com/jyogi-oauth/auth-server/internal/middleware"
	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := store.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	redisClient, err := store.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer redisClient.Close()

	jwtService, err := oauth.NewJWTService(
		cfg.JWTPrivateKeyPath, cfg.JWTPublicKeyPath,
		cfg.JWTKID, cfg.JWTIssuer, cfg.AccessTokenTTL,
	)
	if err != nil {
		log.Fatalf("init jwt service: %v", err)
	}

	memberStore := store.NewMemberStore(pool)
	clientStore := store.NewClientStore(pool)
	scopeStore := store.NewScopeStore(pool)
	consentStore := store.NewConsentStore(pool)
	auditStore := store.NewAuditStore(pool)
	sessionStore := store.NewSessionStore(redisClient, cfg.SessionTTL)
	authCodeStore := store.NewAuthCodeStore(redisClient, cfg.CodeTTL)
	refreshStore := store.NewRefreshStore(redisClient, cfg.RefreshTokenTTL)

	rateLimiter := mw.NewRateLimiter(redisClient)
	sessionMiddleware := mw.NewSessionMiddleware(sessionStore, cfg.SessionCookieName)
	adminMiddleware := mw.NewAdminMiddleware(memberStore)

	pwConfig := &oauth.PasswordConfig{
		Memory:      cfg.Argon2Memory,
		Iterations:  cfg.Argon2Iterations,
		Parallelism: cfg.Argon2Parallelism,
		SaltLength:  16,
		KeyLength:   32,
	}

	jwksHandler := handler.NewJWKSHandler(jwtService)
	adminMemberHandler := handler.NewAdminMemberHandler(memberStore, auditStore, pwConfig)
	loginHandler := handler.NewLoginHandler(
		memberStore, sessionStore, auditStore, rateLimiter,
		cfg.SessionCookieName, cfg.SessionCookieSecure, cfg.SessionCookieDomain,
		cfg.SessionTTL, cfg.RateLimitLogin,
	)
	authorizeHandler := handler.NewAuthorizeHandler(
		clientStore, consentStore, sessionStore, authCodeStore, scopeStore, auditStore, cfg,
	)
	consentHandler := handler.NewConsentHandler(
		sessionStore, consentStore, authCodeStore, clientStore, scopeStore, auditStore, cfg,
	)
	tokenHandler := handler.NewTokenHandler(
		authCodeStore, refreshStore, clientStore, memberStore, jwtService, auditStore, cfg,
	)
	revokeHandler := handler.NewRevokeHandler(refreshStore, clientStore, auditStore)
	introspectHandler := handler.NewIntrospectHandler(jwtService)
	userinfoHandler := handler.NewUserInfoHandler(memberStore, jwtService)
	adminClientHandler := handler.NewAdminClientHandler(clientStore, auditStore)
	meHandler := handler.NewMeHandler(memberStore, sessionStore, pool, cfg.SessionCookieName)
	meIdentityHandler := handler.NewMeIdentityHandler(pool)
	meConsentHandler := handler.NewMeConsentHandler(consentStore, auditStore, pool)
	mePasswordHandler := handler.NewMePasswordHandler(memberStore, auditStore, pwConfig)
	meClientHandler := handler.NewMeClientHandler(clientStore, auditStore)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /oauth/health", func(w http.ResponseWriter, r *http.Request) {
		pgErr := pool.Ping(r.Context())
		redisErr := redisClient.Ping(r.Context()).Err()

		pgStatus := "connected"
		if pgErr != nil {
			pgStatus = "disconnected"
		}
		redisStatus := "connected"
		if redisErr != nil {
			redisStatus = "disconnected"
		}

		status := "healthy"
		code := http.StatusOK
		if pgErr != nil || redisErr != nil {
			status = "unhealthy"
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		fmt.Fprintf(w, `{"status":"%s","service":"auth-server","dependencies":{"postgresql":"%s","redis":"%s"}}`,
			status, pgStatus, redisStatus)
	})

	mux.Handle("GET /oauth/jwks", jwksHandler)
	mux.HandleFunc("POST /oauth/login", loginHandler.Login)
	mux.HandleFunc("GET /oauth/authorize", authorizeHandler.Authorize)
	mux.HandleFunc("GET /oauth/consent", consentHandler.Info)
	mux.HandleFunc("POST /oauth/consent", consentHandler.Process)
	mux.HandleFunc("POST /oauth/token", tokenHandler.Token)
	mux.HandleFunc("POST /oauth/revoke", revokeHandler.Revoke)
	mux.HandleFunc("POST /oauth/introspect", introspectHandler.Introspect)
	mux.HandleFunc("GET /oauth/userinfo", userinfoHandler.UserInfo)

	mux.HandleFunc("GET /oauth/me", meHandler.GetMe)
	mux.HandleFunc("GET /oauth/me/identity", meIdentityHandler.Get)
	mux.HandleFunc("PUT /oauth/me/identity", meIdentityHandler.Update)
	mux.HandleFunc("PUT /oauth/me/password", mePasswordHandler.ChangePassword)
	mux.HandleFunc("GET /oauth/me/consents", meConsentHandler.List)
	mux.HandleFunc("DELETE /oauth/me/consents/{client_id}", meConsentHandler.Revoke)
	mux.HandleFunc("POST /oauth/logout", meHandler.Logout)

	mux.HandleFunc("GET /oauth/me/clients", meClientHandler.List)
	mux.HandleFunc("POST /oauth/me/clients", meClientHandler.Create)
	mux.HandleFunc("PUT /oauth/me/clients/{id}", meClientHandler.Update)
	mux.HandleFunc("DELETE /oauth/me/clients/{id}", meClientHandler.Delete)

	adminMembers := http.NewServeMux()
	adminMembers.HandleFunc("GET /oauth/admin/members", adminMemberHandler.List)
	adminMembers.HandleFunc("POST /oauth/admin/members", adminMemberHandler.Create)
	adminMembers.HandleFunc("GET /oauth/admin/members/{id}", adminMemberHandler.Get)
	adminMembers.HandleFunc("PUT /oauth/admin/members/{id}", adminMemberHandler.Update)
	adminMembers.HandleFunc("DELETE /oauth/admin/members/{id}", adminMemberHandler.Delete)
	mux.Handle("/oauth/admin/members", adminMiddleware.RequireModerator(adminMembers))
	mux.Handle("/oauth/admin/members/", adminMiddleware.RequireModerator(adminMembers))

	adminClients := http.NewServeMux()
	adminClients.HandleFunc("GET /oauth/admin/clients", adminClientHandler.List)
	adminClients.HandleFunc("POST /oauth/admin/clients", adminClientHandler.Create)
	adminClients.HandleFunc("GET /oauth/admin/clients/{id}", adminClientHandler.Get)
	adminClients.HandleFunc("PUT /oauth/admin/clients/{id}", adminClientHandler.Update)
	adminClients.HandleFunc("DELETE /oauth/admin/clients/{id}", adminClientHandler.Delete)
	mux.Handle("/oauth/admin/clients", adminMiddleware.RequireAdmin(adminClients))
	mux.Handle("/oauth/admin/clients/", adminMiddleware.RequireAdmin(adminClients))

	wrappedMux := sessionMiddleware.Wrap(mux)

	addr := cfg.Host + ":" + cfg.Port
	server := &http.Server{
		Addr:         addr,
		Handler:      wrappedMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		server.Shutdown(shutdownCtx)
	}()

	log.Printf("Auth server starting on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
