// Command admin grants and revokes the admin role.
//
// Privilege is deliberately not obtainable over HTTP. There is no "become an
// admin" endpoint and no seeded admin account; the only way to create one is to
// run this tool against the database, which means an operator with database
// credentials made a deliberate decision.
//
//	go run cmd/admin/main.go -list
//	go run cmd/admin/main.go -promote alice@college.edu
//	go run cmd/admin/main.go -demote alice@college.edu
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"lostfound/internal/config"
	"lostfound/internal/database"
)

func main() {
	var (
		promote = flag.String("promote", "", "email address of a user to grant the admin role")
		demote  = flag.String("demote", "", "email address of an admin to return to the user role")
		list    = flag.Bool("list", false, "list current admins")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: admin [-list] [-promote EMAIL] [-demote EMAIL]\n\n")
		fmt.Fprintf(os.Stderr, "The user must have signed in at least once, so that a row exists.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	chosen := 0
	for _, set := range []bool{*promote != "", *demote != "", *list} {
		if set {
			chosen++
		}
	}
	if chosen != 1 {
		flag.Usage()
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		fatalf("failed to load configuration: %v", err)
	}

	ctx := context.Background()
	db, err := database.NewConnection(cfg)
	if err != nil {
		fatalf("failed to connect to the database: %v", err)
	}
	defer db.Close()

	repo := database.NewRepository(db)

	switch {
	case *list:
		admins, err := repo.ListUsersByRole(ctx, "admin")
		if err != nil {
			fatalf("failed to list admins: %v", err)
		}
		if len(admins) == 0 {
			fmt.Println("No admins. Grant one with: admin -promote EMAIL")
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "EMAIL\tNAME\tACTIVE\tSINCE")
		for _, a := range admins {
			fmt.Fprintf(w, "%s\t%s\t%t\t%s\n", a.Email, a.Name, a.IsActive, a.CreatedAt.Format("2006-01-02"))
		}
		w.Flush()

	case *promote != "":
		setRole(ctx, repo, *promote, "admin")

	case *demote != "":
		setRole(ctx, repo, *demote, "user")
	}
}

func setRole(ctx context.Context, repo *database.Repository, email, role string) {
	user, err := repo.SetUserRole(ctx, email, role)
	switch {
	case errors.Is(err, database.ErrNotFound):
		fatalf("no user with email %q. Ask them to sign in once, then run this again.", email)
	case errors.Is(err, database.ErrSystemAccount):
		fatalf("%q is the built-in account used for anonymous posts; its role cannot be changed.", email)
	case err != nil:
		fatalf("failed to set role: %v", err)
	}
	fmt.Printf("%s (%s) now has role %q\n", user.Email, user.Name, user.Role)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "admin: "+format+"\n", args...)
	os.Exit(1)
}
