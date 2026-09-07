package repositories

import (
 "context"
 "time"
 "github.com/alumasinde/tuma254-api/internal/identity/models"
 "github.com/google/uuid"
 "github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
 CreateUser(context.Context,string,string,string,string,string)(models.User,error)
 FindByEmail(context.Context,string)(models.User,string,error)
 FindByID(context.Context,uuid.UUID)(models.User,error)
 CreateSession(context.Context,uuid.UUID,[]byte,time.Time,string,string) error
 ConsumeSession(context.Context,[]byte)(models.User,error)
 RevokeSession(context.Context,[]byte) error
}
type Postgres struct{db *pgxpool.Pool}
func New(db *pgxpool.Pool)*Postgres{return &Postgres{db:db}}
