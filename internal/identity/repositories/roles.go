package repositories

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Postgres) assignRole(ctx context.Context, tx pgx.Tx, userID uuid.UUID, roleName string) error {
	_,err:=tx.Exec(ctx,"INSERT INTO user_roles(user_id,role_id) SELECT $1,id FROM roles WHERE name=$2",userID,roleName)
	return err
}

func (r *Postgres) FindRolesByUserID(ctx context.Context,userID uuid.UUID)([]string,error){
	rows,err:=r.db.Query(ctx,"SELECT ro.name FROM roles ro JOIN user_roles ur ON ur.role_id=ro.id WHERE ur.user_id=$1 ORDER BY ro.name",userID)
	if err!=nil{return nil,err}
	defer rows.Close()
	roles:=make([]string,0)
	for rows.Next(){var role string;if err:=rows.Scan(&role);err!=nil{return nil,err};roles=append(roles,role)}
	return roles,rows.Err()
}
