// Package config reads service configuration from the environment.
package config

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Env returns the value of key, or def when the variable is unset or empty.
func Env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// MustEnv returns the value of key or exits when it is unset.
func MustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "missing required environment variable %s\n", key)
		os.Exit(1)
	}
	return v
}

// DSN builds a Postgres connection string for a single service database from
// the shared DB_* variables. Each service owns its own database.
func DSN(database string) string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(Env("DB_USER", "postgres"), os.Getenv("DB_PASSWORD")),
		Host:   Env("DB_HOST", "localhost") + ":" + Env("DB_PORT", "5432"),
		Path:   "/" + database,
	}
	q := url.Values{}
	q.Set("sslmode", Env("DB_SSLMODE", "disable"))
	u.RawQuery = q.Encode()
	return u.String()
}

// Postgres opens a connection pool for the named service database and verifies
// it with a ping, so a misconfigured service fails at startup and not on the
// first request.
func Postgres(ctx context.Context, database string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, DSN(database))
	if err != nil {
		return nil, fmt.Errorf("connect postgres %s: %w", database, err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres %s: %w", database, err)
	}
	return pool, nil
}

// Redis opens a Redis client and verifies it with a ping.
func Redis(ctx context.Context) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{Addr: Env("REDIS_ADDR", "localhost:6379")})
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}
