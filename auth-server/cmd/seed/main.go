package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jyogi-oauth/auth-server/internal/oauth"
	"github.com/jyogi-oauth/auth-server/internal/store"
)

func main() {
	username := flag.String("username", "", "Admin username")
	password := flag.String("password", "", "Admin password")
	email := flag.String("email", "", "Admin email")
	flag.Parse()

	if *username == "" || *password == "" || *email == "" {
		fmt.Fprintln(os.Stderr, "Usage: seed -username <user> -password <pass> -email <email>")
		os.Exit(1)
	}

	dbURL := os.Getenv("AUTH_DATABASE_URL")
	if dbURL == "" {
		log.Fatal("AUTH_DATABASE_URL is required")
	}

	ctx := context.Background()

	pool, err := store.NewPostgresPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	if err := oauth.ValidatePassword(*password, *username); err != nil {
		log.Fatalf("invalid password: %v", err)
	}

	hash, err := oauth.HashPassword(*password, oauth.DefaultPasswordConfig())
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	memberStore := store.NewMemberStore(pool)
	member, err := memberStore.Create(ctx, *username, hash, *email)
	if err != nil {
		log.Fatalf("create member: %v", err)
	}

	fmt.Printf("Admin member created:\n  ID: %s\n  Username: %s\n  Email: %s\n", member.ID, member.Username, member.Email)
}
