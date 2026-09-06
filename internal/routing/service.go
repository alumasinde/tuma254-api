package routing

import "context"

type Service struct{router Router}
func NewService(router Router)*Service{return &Service{router:router}}
func(s *Service)Route(ctx context.Context,r RouteRequest)(Route,error){return s.router.Route(ctx,r)}
func(s *Service)Matrix(ctx context.Context,r MatrixRequest)([]MatrixCell,error){return s.router.Matrix(ctx,r)}
