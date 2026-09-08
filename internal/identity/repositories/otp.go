package repositories

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/alumasinde/tuma254-api/internal/identity/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Postgres) IssueOTP(ctx context.Context,p IssueOTPParams)error{
	tx,err:=r.db.Begin(ctx);if err!=nil{return err};defer tx.Rollback(ctx)
	var exists bool
	if err=tx.QueryRow(ctx,"SELECT TRUE FROM users WHERE id=$1 FOR UPDATE",p.UserID).Scan(&exists);err!=nil{return err}
	var createdAt time.Time
	err=tx.QueryRow(ctx,"SELECT created_at FROM otp_challenges WHERE user_id=$1 AND purpose=$2 ORDER BY created_at DESC LIMIT 1 FOR UPDATE",p.UserID,p.Purpose).Scan(&createdAt)
	if err!=nil&&!errors.Is(err,pgx.ErrNoRows){return err}
	if err==nil&&time.Since(createdAt)<p.Cooldown{return ErrOTPCooldown}
	var count int
	if err=tx.QueryRow(ctx,"SELECT count(*) FROM otp_challenges WHERE user_id=$1 AND purpose=$2 AND created_at>=now()-($3*interval '1 second')",p.UserID,p.Purpose,int64(p.ResendWindow.Seconds())).Scan(&count);err!=nil{return err}
	if count>=p.MaxResends{return ErrOTPRateLimited}
	if _,err=tx.Exec(ctx,"UPDATE otp_challenges SET revoked_at=now() WHERE user_id=$1 AND purpose=$2 AND verified_at IS NULL AND revoked_at IS NULL",p.UserID,p.Purpose);err!=nil{return err}
	if _,err=tx.Exec(ctx,"INSERT INTO otp_challenges(user_id,phone,purpose,code_hash,expires_at,max_attempts) VALUES($1,$2,$3,$4,$5,$6)",p.UserID,p.Phone,p.Purpose,p.CodeHash,p.ExpiresAt,p.MaxAttempts);err!=nil{return err}
	return tx.Commit(ctx)
}

func (r *Postgres) RevokeActiveOTP(ctx context.Context,userID uuid.UUID,purpose string)error{_,err:=r.db.Exec(ctx,"UPDATE otp_challenges SET revoked_at=now() WHERE user_id=$1 AND purpose=$2 AND verified_at IS NULL AND revoked_at IS NULL",userID,purpose);return err}

func (r *Postgres) VerifyOTP(ctx context.Context,phone,purpose string,codeHash []byte)(models.OTPVerifyResult,error){
	tx,err:=r.db.Begin(ctx);if err!=nil{return models.OTPVerifyResult{},err};defer tx.Rollback(ctx)
	var id,userID uuid.UUID;var stored []byte;var expires time.Time;var attempts,max int
	err=tx.QueryRow(ctx,"SELECT id,user_id,code_hash,expires_at,attempt_count,max_attempts FROM otp_challenges WHERE phone=$1 AND purpose=$2 AND verified_at IS NULL AND revoked_at IS NULL ORDER BY created_at DESC LIMIT 1 FOR UPDATE",phone,purpose).Scan(&id,&userID,&stored,&expires,&attempts,&max)
	if errors.Is(err,pgx.ErrNoRows){return models.OTPVerifyResult{},nil};if err!=nil{return models.OTPVerifyResult{},err}
	if !time.Now().Before(expires){_,err=tx.Exec(ctx,"UPDATE otp_challenges SET revoked_at=now() WHERE id=$1",id);if err!=nil{return models.OTPVerifyResult{},err};if err=tx.Commit(ctx);err!=nil{return models.OTPVerifyResult{},err};return models.OTPVerifyResult{Expired:true},nil}
	if subtle.ConstantTimeCompare(stored,codeHash)!=1{attempts++;exhausted:=attempts>=max;if exhausted{_,err=tx.Exec(ctx,"UPDATE otp_challenges SET attempt_count=$2,revoked_at=now() WHERE id=$1",id,attempts)}else{_,err=tx.Exec(ctx,"UPDATE otp_challenges SET attempt_count=$2 WHERE id=$1",id,attempts)};if err!=nil{return models.OTPVerifyResult{},err};if err=tx.Commit(ctx);err!=nil{return models.OTPVerifyResult{},err};return models.OTPVerifyResult{Exhausted:exhausted},nil}
	if _,err=tx.Exec(ctx,"UPDATE otp_challenges SET verified_at=now() WHERE id=$1",id);err!=nil{return models.OTPVerifyResult{},err}
	if _,err=tx.Exec(ctx,"UPDATE users SET phone_verified_at=COALESCE(phone_verified_at,now()),verification_required=FALSE,is_active=TRUE,updated_at=now() WHERE id=$1 AND phone=$2",userID,phone);err!=nil{return models.OTPVerifyResult{},err}
	var u models.User
	if err=tx.QueryRow(ctx,"SELECT id,email,phone,first_name,last_name,is_active,phone_verified_at,created_at FROM users WHERE id=$1",userID).Scan(&u.ID,&u.Email,&u.Phone,&u.FirstName,&u.LastName,&u.Active,&u.PhoneVerifiedAt,&u.CreatedAt);err!=nil{return models.OTPVerifyResult{},err}
	rows,err:=tx.Query(ctx,"SELECT ro.name FROM roles ro JOIN user_roles ur ON ur.role_id=ro.id WHERE ur.user_id=$1 ORDER BY ro.name",userID);if err!=nil{return models.OTPVerifyResult{},err};defer rows.Close()
	for rows.Next(){var role string;if err=rows.Scan(&role);err!=nil{return models.OTPVerifyResult{},err};u.Roles=append(u.Roles,role)};if err=rows.Err();err!=nil{return models.OTPVerifyResult{},err}
	if err=tx.Commit(ctx);err!=nil{return models.OTPVerifyResult{},err};return models.OTPVerifyResult{User:u,Verified:true},nil
}
