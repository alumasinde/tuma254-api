package main

import(
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alumasinde/tuma254-api/internal/identity"
	"github.com/alumasinde/tuma254-api/internal/platform/config"
	"github.com/alumasinde/tuma254-api/internal/platform/database/mongodb"
	httpserver "github.com/alumasinde/tuma254-api/internal/platform/http"
	"github.com/alumasinde/tuma254-api/internal/platform/logging"
)

func main(){
	cfg,err:=config.Load();if err!=nil{log.Fatal(err)}
	logger:=logging.New(cfg.LogLevel)
	ctx,cancel:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer cancel()
	db,err:=mongodb.Connect(ctx,cfg.MongoDBURI,cfg.MongoDBDatabase)
	if err!=nil{logger.Error("database connection failed","error",err);os.Exit(1)}
	defer db.Close(context.Background())

	repo:=identity.NewRepository(db.Database())
	tokens:=identity.NewTokenManager(cfg.JWTSigningKey,cfg.AccessTokenTTL)
	auth:=identity.NewHandler(identity.NewService(repo,tokens,cfg.RefreshTokenTTL))

	server:=&http.Server{Addr:cfg.HTTPAddr,Handler:httpserver.NewHandler(func()error{return db.Health(context.Background())},auth),ReadHeaderTimeout:5*time.Second}
	go func(){logger.Info("http server started","addr",cfg.HTTPAddr);if err:=server.ListenAndServe();err!=nil&&err!=http.ErrServerClosed{logger.Error("http server failed","error",err);os.Exit(1)}}()
	<-ctx.Done()
	shutdownCtx,shutdownCancel:=context.WithTimeout(context.Background(),10*time.Second);defer shutdownCancel()
	_=server.Shutdown(shutdownCtx)
}
