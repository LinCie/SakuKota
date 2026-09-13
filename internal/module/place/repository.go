package place

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	GetMany(ctx context.Context) ([]Place, error)
	GetCategories(ctx context.Context) ([]Category, error)
	GetManyByCategory(ctx context.Context, categoryID int64) ([]Place, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

var _ Repository = (*PostgresRepository)(nil)

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool: pool,
	}
}

// -----------------------------------------------------------------------------
// PostgresRepository
// -----------------------------------------------------------------------------

func (r *PostgresRepository) GetMany(ctx context.Context) ([]Place, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, sub_district_id, name, address, location,
			operating_hours, estimated_price_range, short_description
		FROM PLACES`,
	)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Place, error) {
		return scanPlace(ctx, row)
	})
}

func (r *PostgresRepository) GetCategories(ctx context.Context) ([]Category, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT id, name, slug FROM CATEGORIES`,
	)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Category, error) {
		return scanCategory(ctx, row)
	})
}

func (r *PostgresRepository) GetManyByCategory(ctx context.Context, categoryID int64) ([]Place, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT p.id, p.sub_district_id, p.name, p.address, p.location,
			p.operating_hours, p.estimated_price_range, p.short_description
		FROM PLACES p
		JOIN PLACE_CATEGORIES pc ON pc.place_id = p.id
		WHERE pc.category_id = $1`,
		categoryID,
	)
	if err != nil {
		return nil, err
	}

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Place, error) {
		return scanPlace(ctx, row)
	})
}

// -----------------------------------------------------------------------------
// Scans
// -----------------------------------------------------------------------------

func scanPlace(ctx context.Context, row pgx.CollectableRow) (Place, error) {
	if err := ctx.Err(); err != nil {
		return Place{}, err
	}

	var p Place

	err := row.Scan(
		&p.ID,
		&p.SubDistrictID,
		&p.Name,
		&p.Address,
		&p.Location,
		&p.OperatingHours,
		&p.EstimatedPriceRange,
		&p.ShortDescription,
	)

	return p, err
}

func scanCategory(ctx context.Context, row pgx.CollectableRow) (Category, error) {
	if err := ctx.Err(); err != nil {
		return Category{}, err
	}

	var c Category

	err := row.Scan(&c.ID, &c.Name, &c.Slug)

	return c, err
}
