// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

// Package admin implements administrative endpoints for satellite.
package admin

import (
	"context"
	"crypto/subtle"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"storj.io/common/errs2"
	"storj.io/storj/satellite/accounting"
	"storj.io/storj/satellite/console"
	"storj.io/storj/satellite/metainfo"
	"storj.io/storj/satellite/payments"
	"storj.io/storj/satellite/payments/stripecoinpayments"
)

var (
	//go:embed ui/public
	ui embed.FS
	// uiAssets has the ui files but without the path prefix '/ui/public'.
	// The variable is initialized in the init function.
	uiAssets fs.FS
)

func init() {
	var err error
	uiAssets, err = fs.Sub(ui, "ui/public")
	if err != nil {
		panic(fmt.Sprintf("invalid ui assets, they should have the content under '/ui/public' directory path. %+v", err))
	}
}

// Config defines configuration for debug server.
type Config struct {
	Address string `help:"admin peer http listening address" releaseDefault:"" devDefault:""`

	AuthorizationToken string `internal:"true"`
}

// DB is databases needed for the admin server.
type DB interface {
	// ProjectAccounting returns database for storing information about project data use
	ProjectAccounting() accounting.ProjectAccounting
	// Console returns database for satellite console
	Console() console.DB
	// StripeCoinPayments returns database for satellite stripe coin payments
	StripeCoinPayments() stripecoinpayments.DB
	// Buckets returns database for satellite buckets
	Buckets() metainfo.BucketsDB
}

// Server provides endpoints for administrative tasks.
type Server struct {
	log *zap.Logger

	listener net.Listener
	server   http.Server

	db       DB
	payments payments.Accounts

	nowFn func() time.Time
}

// NewServer returns a new administration Server.
func NewServer(log *zap.Logger, listener net.Listener, db DB, accounts payments.Accounts, config Config) *Server {
	server := &Server{
		log: log,

		listener: listener,

		db:       db,
		payments: accounts,

		nowFn: time.Now,
	}

	root := mux.NewRouter()

	api := root.PathPrefix("/api/").Subrouter()
	api.Use((&protectedServer{
		allowedAuthorization: config.AuthorizationToken,
	}).Middleware)

	// When adding new options, also update README.md
	api.HandleFunc("/users", server.addUser).Methods("POST")
	api.HandleFunc("/users/{useremail}", server.updateUser).Methods("PUT")
	api.HandleFunc("/users/{useremail}", server.userInfo).Methods("GET")
	api.HandleFunc("/users/{useremail}", server.deleteUser).Methods("DELETE")
	api.HandleFunc("/coupons", server.addCoupon).Methods("POST")
	api.HandleFunc("/coupons/{couponid}", server.couponInfo).Methods("GET")
	api.HandleFunc("/coupons/{couponid}", server.deleteCoupon).Methods("DELETE")
	api.HandleFunc("/projects", server.addProject).Methods("POST")
	api.HandleFunc("/projects/{project}/usage", server.checkProjectUsage).Methods("GET")
	api.HandleFunc("/projects/{project}/limit", server.getProjectLimit).Methods("GET")
	api.HandleFunc("/projects/{project}/limit", server.putProjectLimit).Methods("PUT", "POST")
	api.HandleFunc("/projects/{project}", server.getProject).Methods("GET")
	api.HandleFunc("/projects/{project}", server.renameProject).Methods("PUT")
	api.HandleFunc("/projects/{project}", server.deleteProject).Methods("DELETE")
	api.HandleFunc("/projects/{project}/apikeys", server.listAPIKeys).Methods("GET")
	api.HandleFunc("/projects/{project}/apikeys", server.addAPIKey).Methods("POST")
	api.HandleFunc("/projects/{project}/apikeys/{name}", server.deleteAPIKeyByName).Methods("DELETE")
	api.HandleFunc("/apikeys/{apikey}", server.deleteAPIKey).Methods("DELETE")

	// This handler must be the last one because it uses the root as prefix,
	// otherwise will try to serve all the handlers set after this one.
	root.PathPrefix("/").Handler(http.FileServer(http.FS(uiAssets))).Methods("GET")

	server.server.Handler = root
	return server
}

type protectedServer struct {
	allowedAuthorization string
}

func (server *protectedServer) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if server.allowedAuthorization == "" {
			sendJSONError(w, "Authorization not enabled.",
				"", http.StatusForbidden)
			return
		}

		equality := subtle.ConstantTimeCompare(
			[]byte(r.Header.Get("Authorization")),
			[]byte(server.allowedAuthorization),
		)
		if equality != 1 {
			sendJSONError(w, "Forbidden",
				"", http.StatusForbidden)
			return
		}

		r.Header.Set("Cache-Control", "must-revalidate")
		next.ServeHTTP(w, r)
	})
}

// Run starts the admin endpoint.
func (server *Server) Run(ctx context.Context) error {
	if server.listener == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	var group errgroup.Group
	group.Go(func() error {
		<-ctx.Done()
		return Error.Wrap(server.server.Shutdown(context.Background()))
	})
	group.Go(func() error {
		defer cancel()
		err := server.server.Serve(server.listener)
		if errs2.IsCanceled(err) || errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		return Error.Wrap(err)
	})
	return group.Wait()
}

// SetNow allows tests to have the server act as if the current time is whatever they want.
func (server *Server) SetNow(nowFn func() time.Time) {
	server.nowFn = nowFn
}

// Close closes server and underlying listener.
func (server *Server) Close() error {
	return Error.Wrap(server.server.Close())
}
