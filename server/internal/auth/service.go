package auth

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/John-DengD/smu-deal/server/internal/db/gen"
	"github.com/John-DengD/smu-deal/server/internal/httpx"
)

// Querier is the subset of the sqlc-generated *gen.Queries the auth service needs.
type Querier interface {
	GetUserByStudentNo(ctx context.Context, studentNo string) (gen.User, error)
	GetUserByID(ctx context.Context, id int64) (gen.User, error)
	CountByStudentNo(ctx context.Context, studentNo string) (int64, error)
	InsertUser(ctx context.Context, arg gen.InsertUserParams) (gen.User, error)
	UpdateUserProfile(ctx context.Context, arg gen.UpdateUserProfileParams) (gen.User, error)
}

type Service struct {
	q   Querier
	jwt httpx.JWT
}

func NewService(q Querier, jwt httpx.JWT) *Service {
	return &Service{q: q, jwt: jwt}
}

// UserInfo mirrors AuthDTO.UserInfo (camelCase wire contract).
type UserInfo struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	StudentNo string  `json:"studentNo"`
	Phone     *string `json:"phone"`
	College   *string `json:"college"`
	Campus    *string `json:"campus"`
	Avatar    *string `json:"avatar"`
	Role      string  `json:"role"`
	Status    string  `json:"status"`
}

// LoginResp mirrors AuthDTO.LoginResp.
type LoginResp struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

type RegisterReq struct {
	Name      string  `json:"name"`
	StudentNo string  `json:"studentNo"`
	Password  string  `json:"password"`
	Phone     *string `json:"phone"`
	College   *string `json:"college"`
	Campus    *string `json:"campus"`
}

type LoginReq struct {
	StudentNo string `json:"studentNo"`
	Password  string `json:"password"`
}

func toInfo(u gen.User) UserInfo {
	return UserInfo{
		ID:        u.ID,
		Name:      u.Name,
		StudentNo: u.StudentNo,
		Phone:     u.Phone,
		College:   u.College,
		Campus:    u.Campus,
		Avatar:    u.Avatar,
		Role:      u.Role,
		Status:    u.Status,
	}
}

func (s *Service) Register(ctx context.Context, req RegisterReq) (UserInfo, error) {
	if strings.TrimSpace(req.Name) == "" {
		return UserInfo{}, httpx.Biz("请输入姓名")
	}
	if strings.TrimSpace(req.StudentNo) == "" {
		return UserInfo{}, httpx.Biz("请输入学号")
	}
	if req.Password == "" {
		return UserInfo{}, httpx.Biz("请输入密码")
	}
	if pw := utf8.RuneCountInString(req.Password); pw < 6 || pw > 64 {
		return UserInfo{}, httpx.Biz("密码长度需为 6-64 位")
	}

	studentNo, err := validateStudentNo(req.StudentNo)
	if err != nil {
		return UserInfo{}, err
	}
	name, err := validateName(req.Name)
	if err != nil {
		return UserInfo{}, err
	}

	cnt, err := s.q.CountByStudentNo(ctx, studentNo)
	if err != nil {
		return UserInfo{}, err
	}
	if cnt > 0 {
		return UserInfo{}, httpx.Biz("学号已被注册")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return UserInfo{}, err
	}

	u, err := s.q.InsertUser(ctx, gen.InsertUserParams{
		Name:         name,
		StudentNo:    studentNo,
		PasswordHash: string(hash),
		Phone:        req.Phone,
		College:      req.College,
		Campus:       req.Campus,
		Role:         "USER",
		Status:       "ACTIVE",
	})
	if err != nil {
		return UserInfo{}, err
	}
	return toInfo(u), nil
}

func (s *Service) Login(ctx context.Context, req LoginReq) (LoginResp, error) {
	u, err := s.q.GetUserByStudentNo(ctx, req.StudentNo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LoginResp{}, httpx.Biz("账号或密码错误")
		}
		return LoginResp{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		return LoginResp{}, httpx.Biz("账号或密码错误")
	}
	if strings.EqualFold(u.Status, "DISABLED") {
		return LoginResp{}, httpx.Biz("账号已被禁用")
	}
	token, err := s.jwt.Generate(u.ID, u.Role)
	if err != nil {
		return LoginResp{}, err
	}
	return LoginResp{Token: token, User: toInfo(u)}, nil
}

func (s *Service) GetMe(ctx context.Context, id int64) (UserInfo, error) {
	u, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserInfo{}, httpx.NewBiz(401, "用户不存在")
		}
		return UserInfo{}, err
	}
	return toInfo(u), nil
}

func (s *Service) UpdateMe(ctx context.Context, id int64, req UserInfo) (UserInfo, error) {
	u, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserInfo{}, httpx.NewBiz(401, "用户不存在")
		}
		return UserInfo{}, err
	}

	name := u.Name
	if req.Name != "" {
		vn, verr := validateName(req.Name)
		if verr != nil {
			return UserInfo{}, verr
		}
		name = vn
	}
	phone := u.Phone
	if req.Phone != nil {
		phone = req.Phone
	}
	college := u.College
	if req.College != nil {
		college = req.College
	}
	campus := u.Campus
	if req.Campus != nil {
		campus = req.Campus
	}
	avatar := u.Avatar
	if req.Avatar != nil {
		avatar = req.Avatar
	}

	updated, err := s.q.UpdateUserProfile(ctx, gen.UpdateUserProfileParams{
		ID:      id,
		Name:    name,
		Phone:   phone,
		College: college,
		Campus:  campus,
		Avatar:  avatar,
	})
	if err != nil {
		return UserInfo{}, err
	}
	return toInfo(updated), nil
}
