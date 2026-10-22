package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	authpostgres "github.com/simple-erp/backend/internal/auth/adapter/postgres"
	"github.com/simple-erp/backend/internal/auth/application"
	"github.com/simple-erp/backend/internal/auth/domain"
)

func main() {
	email := flag.String("email", "", "initial Super Admin email")
	name := flag.String("name", "Administrator", "display name")
	flag.Parse()
	if *email == "" {
		log.Fatal("--email is required")
	}
	if os.Getenv("DATABASE_URL") == "" {
		log.Fatal("DATABASE_URL is required")
	}
	fmt.Fprint(os.Stderr, "Password (read from stdin): ")
	password, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		log.Fatal("could not read password from stdin")
	}
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	svc := application.NewBootstrapService(authpostgres.BootstrapStore{Pool: pool}, domain.NewPasswordHasher())
	if err := svc.Execute(context.Background(), *email, *name, strings.TrimSpace(password)); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Initial Super Admin created.")
}
