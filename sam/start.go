package sam

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/SentimensRG/ctx"
	"github.com/SentimensRG/ctx/sigctx"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/pkg/errors"
	"github.com/titpetric/factory"
	"github.com/titpetric/factory/resputil"

	authService "github.com/crusttech/crust/auth/service"
	"github.com/crusttech/crust/internal/auth"
	"github.com/crusttech/crust/sam/rest"
	samService "github.com/crusttech/crust/sam/service"
	"github.com/crusttech/crust/sam/websocket"
)

func Init() error {
	// validate configuration
	if err := flags.Validate(); err != nil {
		return err
	}

	// start/configure database connection
	factory.Database.Add("default", flags.db.DSN)
	db, err := factory.Database.Get()
	if err != nil {
		return err
	}
	// @todo: profiling as an external service?
	switch flags.db.Profiler {
	case "stdout":
		db.Profiler = &factory.Database.ProfilerStdout
	default:
		fmt.Println("No database query profiler selected")
	}

	// configure resputil options
	resputil.SetConfig(resputil.Options{
		Pretty: flags.http.Pretty,
		Trace:  flags.http.Tracing,
		Logger: func(err error) {
			// @todo: error logging
		},
	})

	authService.Init()
	samService.Init()

	return nil
}

func Start() error {
	deadline := sigctx.New()

	log.Println("Starting http server on address " + flags.http.Addr)
	listener, err := net.Listen("tcp", flags.http.Addr)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Can't listen on addr %s", flags.http.Addr))
	}

	// JWT Auth
	jwtAuth, err := auth.JWT()
	if err != nil {
		return errors.Wrap(err, "Error creating JWT Auth object")
	}

	r := chi.NewRouter()
	r.Use(handleCORS)

	// Only protect application routes with JWT
	r.Group(func(r chi.Router) {
		r.Use(jwtAuth.Verifier(), jwtAuth.Authenticator())
		mountRoutes(r, flags.http, rest.MountRoutes(), websocket.MountRoutes(ctx.AsContext(deadline), flags.repository))
	})

	printRoutes(r, flags.http)
	mountSystemRoutes(r, flags.http)

	go http.Serve(listener, r)
	<-deadline.Done()

	return nil
}

// Sets up default CORS rules to use as a middleware
func handleCORS(next http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}).Handler(next)
}