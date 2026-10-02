package porder

import (
	"context"
	"errors"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/m1ll3r1337/order-service/internal/app/entity"
	"github.com/m1ll3r1337/order-service/internal/app/repository"
	rcpostgres "github.com/m1ll3r1337/order-service/internal/app/repository/conn/postgres"
)

type repoPg struct {
	conn *rcpostgres.Client
}

func NewRepo(client *rcpostgres.Client) repository.Order {
	return &repoPg{conn: client}
}

func (r *repoPg) InsideTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.conn.InsideTx(ctx, fn)
}

func (r *repoPg) Create(ctx context.Context, order entity.Order) error {
	db := r.conn.GetDB(ctx).WithContext(ctx)
	if err := db.Create(&order).Error; err != nil {
		return err
	}
	return nil
}

func (r *repoPg) GetByGUID(ctx context.Context, guid uuid.UUID) (entity.Order, error) {
	db := r.conn.GetDB(ctx).WithContext(ctx)

	var order entity.Order

	if err := db.Preload("Items").Where("guid = ?", guid).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.Order{}, entity.ErrNotFound
		}
		return entity.Order{}, err
	}

	return order, nil
}

func (r *repoPg) Update(ctx context.Context, order entity.Order) error {
	db := r.conn.GetDB(ctx).WithContext(ctx)

	res := db.Model(&entity.Order{}).Where("guid = ?", order.GUID).Updates(map[string]any{
		"status":     order.Status,
		"updated_at": order.UpdatedAt,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return entity.ErrNotFound
	}

	return nil
}

func (r *repoPg) Delete(ctx context.Context, guid uuid.UUID) error {
	db := r.conn.GetDB(ctx).WithContext(ctx)

	res := db.Where("guid = ?", guid).Delete(&entity.Order{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return entity.ErrNotFound
	}

	return nil
}

func (r *repoPg) List(ctx context.Context, status *string, userGUID *uuid.UUID) ([]entity.Order, error) {
	db := r.conn.GetDB(ctx).WithContext(ctx)

	var orders []entity.Order

	query := db.Preload("Items")

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if userGUID != nil {
		query = query.Where("user_guid = ?", *userGUID)
	}

	if err := query.Find(&orders).Error; err != nil {
		return nil, err
	}

	return orders, nil
}
