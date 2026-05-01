package db

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type seedUser struct {
	name          string
	email         string
	password      string
	role          string
	associationID *uuid.UUID
}

func RunSeeds(ctx context.Context, pool *pgxpool.Pool) error {
	if err := seedAssociations(ctx, pool); err != nil {
		return fmt.Errorf("seed associations: %w", err)
	}
	if err := seedUsers(ctx, pool); err != nil {
		return fmt.Errorf("seed users: %w", err)
	}
	return nil
}

func seedAssociations(ctx context.Context, pool *pgxpool.Pool) error {
	associations := []struct {
		name string
		code string
	}{
		{"サンプル自治会", "SAMPLE01"},
		{"テスト自治会", "TEST001"},
	}

	for _, a := range associations {
		var exists bool
		err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM associations WHERE code = $1)`, a.code).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		now := time.Now()
		_, err = pool.Exec(ctx,
			`INSERT INTO associations (id, name, code, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			uuid.New(), a.name, a.code, now, now,
		)
		if err != nil {
			return err
		}
		fmt.Printf("[seed] association created: %s (%s)\n", a.name, a.code)
	}
	return nil
}

func seedUsers(ctx context.Context, pool *pgxpool.Pool) error {
	// SAMPLe01の自治会IDを取得
	var sampleAssocID uuid.UUID
	err := pool.QueryRow(ctx, `SELECT id FROM associations WHERE code = 'SAMPLE01'`).Scan(&sampleAssocID)
	if err != nil {
		return fmt.Errorf("get SAMPLE01 association: %w", err)
	}

	users := []seedUser{
		{
			name:          "システム管理者",
			email:         "admin@system.local",
			password:      "Admin1234!",
			role:          "system_admin",
			associationID: nil,
		},
		{
			name:          "自治会管理者",
			email:         "admin@sample01.local",
			password:      "Admin1234!",
			role:          "association_admin",
			associationID: &sampleAssocID,
		},
		{
			name:          "一般ユーザー",
			email:         "user@sample01.local",
			password:      "User1234!",
			role:          "user",
			associationID: &sampleAssocID,
		},
	}

	for _, u := range users {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, u.email).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password for %s: %w", u.email, err)
		}

		now := time.Now()
		_, err = pool.Exec(ctx,
			`INSERT INTO users (id, association_id, name, email, password_hash, role, is_active, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, true, $7, $8)`,
			uuid.New(), u.associationID, u.name, u.email, string(hash), u.role, now, now,
		)
		if err != nil {
			return fmt.Errorf("insert user %s: %w", u.email, err)
		}
		fmt.Printf("[seed] user created: %s (%s) role=%s\n", u.name, u.email, u.role)
	}
	return nil
}
