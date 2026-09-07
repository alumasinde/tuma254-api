package repositories

import (
 "context"
 "errors"
 "net"
 "strings"
 "time"
 "github.com/alumasinde/tuma254-api/internal/identity/models"
 "github.com/google/uuid"
)

func (r *Postgres) CreateUser(ctx context.Context,email,phone,first,last,hash string)(models.User,error){
 tx,err:=r.db.Begin(ctx);if err!=nil{return models.User{},err};defer tx.Rollback(ctx)
 var u models.User
 err=tx.QueryRow(ctx,`INSERT INTO users(email,phone,password_hash,first_name,last_name) VALUES($1,$2,$3,$4,$5) RETURNING id,email,phone,first_name,last_name,is_active,created_at`,email,phone,hash,first,last).Scan(&u.ID,&u.Email,&u.Phone,&u.FirstName,&u.LastName,&u.Active,&u.CreatedAt);if err!=nil{return models.User{},err}
 _,err=tx.Exec(ctx,`INSERT INTO user_roles(user_id,role_id) SELECT $1,id FROM roles WHERE name='customer'`,u.ID);if err!=nil{return models.User{},err}
 if err=tx.Commit(ctx);err!=nil{return models.User{},err};u.Roles=[]string{"customer"};return u,nil
}
func(r *Postgres)FindByEmail(ctx context.Context,email string)(models.User,string,error){var u models.User;var hash string;err:=r.db.QueryRow(ctx,`SELECT id,email,phone,password_hash,first_name,last_name,is_active,created_at FROM users WHERE lower(email)=lower($1)`,email).Scan(&u.ID,&u.Email,&u.Phone,&hash,&u.FirstName,&u.LastName,&u.Active,&u.CreatedAt);if err!=nil{return models.User{},"",err};rs,err:=r.roles(ctx,u.ID);u.Roles=rs;return u,hash,err}
func(r *Postgres)FindByID(ctx context.Context,id uuid.UUID)(models.User,error){var u models.User;err:=r.db.QueryRow(ctx,`SELECT id,email,phone,first_name,last_name,is_active,created_at FROM users WHERE id=$1`,id).Scan(&u.ID,&u.Email,&u.Phone,&u.FirstName,&u.LastName,&u.Active,&u.CreatedAt);if err!=nil{return models.User{},err};rs,err:=r.roles(ctx,id);u.Roles=rs;return u,err}
func(r *Postgres)roles(ctx context.Context,id uuid.UUID)([]string,error){rows,err:=r.db.Query(ctx,`SELECT ro.name FROM roles ro JOIN user_roles ur ON ur.role_id=ro.id WHERE ur.user_id=$1 ORDER BY ro.name`,id);if err!=nil{return nil,err};defer rows.Close();out:=[]string{};for rows.Next(){var s string;if err=rows.Scan(&s);err!=nil{return nil,err};out=append(out,s)};return out,rows.Err()}
func(r *Postgres)CreateSession(ctx context.Context,id uuid.UUID,h []byte,exp time.Time,ua,ip string)error{var addr any;if p:=net.ParseIP(strings.TrimSpace(ip));p!=nil{addr=p.String()};_,err:=r.db.Exec(ctx,`INSERT INTO refresh_sessions(user_id,token_hash,expires_at,user_agent,ip_address) VALUES($1,$2,$3,$4,$5)`,id,h,exp,ua,addr);return err}
func(r *Postgres)ConsumeSession(ctx context.Context,h []byte)(models.User,error){tx,err:=r.db.Begin(ctx);if err!=nil{return models.User{},err};defer tx.Rollback(ctx);var id uuid.UUID;err=tx.QueryRow(ctx,`UPDATE refresh_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at>now() RETURNING user_id`,h).Scan(&id);if err!=nil{return models.User{},err};var u models.User;err=tx.QueryRow(ctx,`SELECT id,email,phone,first_name,last_name,is_active,created_at FROM users WHERE id=$1`,id).Scan(&u.ID,&u.Email,&u.Phone,&u.FirstName,&u.LastName,&u.Active,&u.CreatedAt);if err!=nil{return models.User{},err};if !u.Active{return models.User{},errors.New("inactive")};rows,err:=tx.Query(ctx,`SELECT ro.name FROM roles ro JOIN user_roles ur ON ur.role_id=ro.id WHERE ur.user_id=$1 ORDER BY ro.name`,id);if err!=nil{return models.User{},err};for rows.Next(){var s string;if err=rows.Scan(&s);err!=nil{rows.Close();return models.User{},err};u.Roles=append(u.Roles,s)};rows.Close();if err=rows.Err();err!=nil{return models.User{},err};if err=tx.Commit(ctx);err!=nil{return models.User{},err};return u,nil}
func(r *Postgres)RevokeSession(ctx context.Context,h []byte)error{_,err:=r.db.Exec(ctx,`UPDATE refresh_sessions SET revoked_at=now() WHERE token_hash=$1 AND revoked_at IS NULL`,h);return err}
