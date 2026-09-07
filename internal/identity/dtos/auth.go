package dtos

type RegisterRequest struct { Email string `json:"email"`; Phone string `json:"phone"`; FirstName string `json:"firstName"`; LastName string `json:"lastName"`; Password string `json:"password"` }
type LoginRequest struct { Email string `json:"email"`; Password string `json:"password"` }
type RefreshRequest struct { RefreshToken string `json:"refreshToken"` }
type AuthResponse struct { AccessToken string `json:"accessToken"`; RefreshToken string `json:"refreshToken"`; TokenType string `json:"tokenType"`; ExpiresIn int64 `json:"expiresIn"` }
type MeResponse struct { ID string `json:"id"`; Email string `json:"email"`; Phone string `json:"phone"`; FirstName string `json:"firstName"`; LastName string `json:"lastName"`; Roles []string `json:"roles"` }
