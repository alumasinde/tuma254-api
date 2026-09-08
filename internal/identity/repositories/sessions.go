package repositories

import (
	"context"
	"net"
	"strings"

	"github.com/alumasinde/tuma254-api/internal/identity/models"
	"github.com/google/uuid"
)

func (r *Postgres) CreateSession(ctx context.Context,p CreateSessionParams)error{_,err:=r.db.Exec(ctx,"INSERT INTO refresh_sessions(user_id,token_hash,expires_at,user_agent,ip_address) VALUES($1,$2,$3,$4,$5)",p.UserID,p.TokenHash,p.ExpiresAt,p.UserAgent,parseIPAddress(p.IPAddress));return err}

func (r *Postgres) RotateSession(ctx context.Context,p RotateSessionParams)(models.User,error){
	tx,err:=r.db.Begin(ctx);if err!=nil{return models.User{},err};defer tx.Rollback()
	var userID uuid.UUID
	if err=tx.QueryRow(ctx,"UPDATE refresh_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>now() RETURNING user_id",p.CurrentTokenHash).Scan(&userID);err!=nil{return models.User{},err}
	var u models.User
	if err=tx.QueryRow(ctx,"SELECT id,email,phone,first_name,last_name,is_active,phone_verified_at,created_at FROM users WHERE id=$1",userID).Scan(&u.ID,&u.Email,&u.Phone,&u.FirstName,&u.LastName,&u.Active,&u.PhoneVerifiedAt,&u.CreatedAt);err!=nil{return models.User{},err}
	if !u.Active{return models.User{},ErrAccountInactive}
	rows,err:=tx.Query(ctx,"SELECT ro.name FROM roles ro JOIN user_roles ur ON ur.role_id=ro.id WHERE ur.user_id=$1 ORDER BY ro.name",userID);if err!=nil{return models.User{},err};defer rows.Close()
	for rows.Next(){var role string;if err=rows.Scan(&role);err!=nil{return models.User{},err};u.Roles=append(u.Roles,role)};if err=rows.Err();err!=nil{return models.User{},err}
	if _,err=tx.Exec(ctx,"INSERT INTO refresh_sessions(user_id,token_hash,expires_at,user_agent,ip_address) VALUES($1,$2,$3,$4,$5)",userID,p.ReplacementTokenHash,p.ReplacementExpiresAt,p.UserAgent,parseIPAddress(p.IPAddress));err!=nil{return models.User{},err}
	if err=tx.Commit(ctx);err!=nil{return models.User{},err};return u,nil
}

func (r *Postgres) RevokeSession(ctx context.Context,tokenHash []byte)error{_,err:=r.db.Exec(ctx,"UPDATE refresh_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL",tokenHash);return err}
func parseIPAddress(value string)any{if parsed:=net.ParseIP(strings.TrimSpace(value));parsed!=nil{return parsed.String()};return nil}
