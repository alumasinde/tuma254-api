package repositories

import (
	"context"

	"github.com/alumasinde/tuma254-api/internal/identity/models"
	"github.com/google/uuid"
)

func (r *Postgres) CreateUser(ctx context.Context, params CreateUserParams) (models.User, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil { return models.User{}, err }
	defer tx.Rollback(ctx)

	var user models.User
	err = tx.QueryRow(ctx, "INSERT INTO users(email,phone,password_hash,first_name,last_name,is_active,verification_required) VALUES($1,$2,$3,$4,$5,FALSE,TRUE) RETURNING id,email,phone,first_name,last_name,is_active,phone_verified_at,created_at", params.Email, params.Phone, params.PasswordHash, params.FirstName, params.LastName).Scan(&user.ID,&user.Email,&user.Phone,&user.FirstName,&user.LastName,&user.Active,&user.PhoneVerifiedAt,&user.CreatedAt)
	if err != nil { return models.User{}, err }
	if err := r.assignRole(ctx, tx, user.ID, "customer"); err != nil { return models.User{}, err }
	if err := tx.Commit(ctx); err != nil { return models.User{}, err }
	user.Roles=[]string{"customer"}
	return user,nil
}

func (r *Postgres) FindByEmail(ctx context.Context, email string) (models.User,string,error) {
	var user models.User
	var passwordHash string
	err:=r.db.QueryRow(ctx,"SELECT id,email,phone,password_hash,first_name,last_name,is_active,phone_verified_at,created_at FROM users WHERE lower(email)=lower($1)",email).Scan(&user.ID,&user.Email,&user.Phone,&passwordHash,&user.FirstName,&user.LastName,&user.Active,&user.PhoneVerifiedAt,&user.CreatedAt)
	if err!=nil{return models.User{},"",err}
	roles,err:=r.FindRolesByUserID(ctx,user.ID); user.Roles=roles
	return user,passwordHash,err
}

func (r *Postgres) FindByPhone(ctx context.Context, phone string) (models.User, error) {
	return r.findUserByPhone(ctx, phone)
}

func (r *Postgres) FindByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	return r.findUserByID(ctx, id)
}

func (r *Postgres) findUserByPhone(ctx context.Context, phone string) (models.User, error) {
	return r.scanUser(ctx, "SELECT id,email,phone,first_name,last_name,is_active,phone_verified_at,created_at FROM users WHERE phone=$1", phone)
}

func (r *Postgres) findUserByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	return r.scanUser(ctx, "SELECT id,email,phone,first_name,last_name,is_active,phone_verified_at,created_at FROM users WHERE id=$1", id)
}

func (r *Postgres) scanUser(ctx context.Context, query string, value any) (models.User, error) {
	var user models.User
	err := r.db.QueryRow(ctx, query, value).Scan(&user.ID, &user.Email, &user.Phone, &user.FirstName, &user.LastName, &user.Active, &user.PhoneVerifiedAt, &user.CreatedAt)
	if err != nil {
		return models.User{}, err
	}
	roles, err := r.FindRolesByUserID(ctx, user.ID)
	if err != nil {
		return models.User{}, err
	}
	user.Roles = roles
	return user, nil
}
