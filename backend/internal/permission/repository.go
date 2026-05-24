package permission

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository は権限管理のDB操作を担う
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetRoles はロール一覧を取得する
func (r *Repository) GetRoles(ctx context.Context) ([]*RoleModel, error) {
	const q = `
		SELECT id, name, display_name, is_system, created_at, updated_at
		FROM roles
		ORDER BY CASE name
			WHEN 'system_admin'      THEN 1
			WHEN 'association_admin' THEN 2
			WHEN 'vice_admin'        THEN 3
			WHEN 'user_admin'        THEN 4
			WHEN 'user'              THEN 5
			ELSE 9
		END`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*RoleModel
	for rows.Next() {
		rm := &RoleModel{}
		if err := rows.Scan(&rm.ID, &rm.Name, &rm.DisplayName, &rm.IsSystem, &rm.CreatedAt, &rm.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, rm)
	}
	return roles, rows.Err()
}

// GetFeatures は機能一覧を取得する
func (r *Repository) GetFeatures(ctx context.Context) ([]*FeatureModel, error) {
	const q = `SELECT id, name, display_name, sort_order FROM features ORDER BY sort_order`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []*FeatureModel
	for rows.Next() {
		fm := &FeatureModel{}
		if err := rows.Scan(&fm.ID, &fm.Name, &fm.DisplayName, &fm.SortOrder); err != nil {
			return nil, err
		}
		features = append(features, fm)
	}
	return features, rows.Err()
}

// GetAllPermissions は権限マトリクスを取得する。
// assocID が nil の場合はデフォルト設定（association_id IS NULL）のみを返す。
// assocID が指定された場合は自治会専用設定を優先し、存在しなければ
// デフォルト設定にフォールバックした有効な権限一覧を返す。
// 各行の AssociationID が非 nil の場合は自治会専用設定（IsCustomized=true）、
// nil の場合はデフォルト設定のフォールバック（IsCustomized=false）を意味する。
func (r *Repository) GetAllPermissions(ctx context.Context, assocID *uuid.UUID) ([]*RolePermission, error) {
	if assocID == nil {
		return r.getAllPermissionsDefault(ctx)
	}
	return r.getAllPermissionsForAssociation(ctx, *assocID)
}

func (r *Repository) getAllPermissionsDefault(ctx context.Context) ([]*RolePermission, error) {
	const q = `
		SELECT rp.id, rp.role_id, rp.feature_id, rp.association_id,
		       rp.can_view, rp.can_create, rp.can_edit, rp.can_delete,
		       rp.scope, rp.created_at, rp.updated_at
		FROM role_permissions rp
		JOIN roles ro ON ro.id = rp.role_id
		JOIN features f ON f.id = rp.feature_id
		WHERE rp.association_id IS NULL
		ORDER BY f.sort_order, ro.id`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []*RolePermission
	for rows.Next() {
		p := &RolePermission{}
		if err := rows.Scan(
			&p.ID, &p.RoleID, &p.FeatureID, &p.AssociationID,
			&p.CanView, &p.CanCreate, &p.CanEdit, &p.CanDelete,
			&p.Scope, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.IsCustomized = false
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

func (r *Repository) getAllPermissionsForAssociation(ctx context.Context, assocID uuid.UUID) ([]*RolePermission, error) {
	// DISTINCT ON で自治会専用設定を優先してデフォルト設定にフォールバック。
	// ORDER BY (rp.association_id IS NULL) は false=0（自治会専用）を先に並べるため、
	// DISTINCT ON は自治会専用設定がある場合はそれを選択し、なければデフォルトを返す。
	const q = `
		SELECT DISTINCT ON (rp.role_id, rp.feature_id)
		       rp.id, rp.role_id, rp.feature_id, rp.association_id,
		       rp.can_view, rp.can_create, rp.can_edit, rp.can_delete,
		       rp.scope, rp.created_at, rp.updated_at
		FROM role_permissions rp
		JOIN roles ro ON ro.id = rp.role_id
		JOIN features f ON f.id = rp.feature_id
		WHERE rp.association_id = $1 OR rp.association_id IS NULL
		ORDER BY rp.role_id, rp.feature_id, (rp.association_id IS NULL)`
	rows, err := r.pool.Query(ctx, q, assocID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var perms []*RolePermission
	for rows.Next() {
		p := &RolePermission{}
		if err := rows.Scan(
			&p.ID, &p.RoleID, &p.FeatureID, &p.AssociationID,
			&p.CanView, &p.CanCreate, &p.CanEdit, &p.CanDelete,
			&p.Scope, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.IsCustomized = p.AssociationID != nil
		perms = append(perms, p)
	}
	return perms, rows.Err()
}

// GetPermissionByRoleAndFeature はロール名・機能名で権限チェック用データを取得する。
// assocID が nil の場合はデフォルト設定（association_id IS NULL）のみを参照する。
// assocID が指定された場合は自治会専用設定を優先し、存在しなければ
// デフォルト設定にフォールバックした有効な権限を返す。
func (r *Repository) GetPermissionByRoleAndFeature(ctx context.Context, roleName, featureName string, assocID *uuid.UUID) (*PermissionCheck, error) {
	pc := &PermissionCheck{}

	if assocID == nil {
		const q = `
			SELECT rp.can_view, rp.can_create, rp.can_edit, rp.can_delete, rp.scope
			FROM role_permissions rp
			JOIN roles ro ON ro.id = rp.role_id
			JOIN features f ON f.id = rp.feature_id
			WHERE ro.name = $1 AND f.name = $2 AND rp.association_id IS NULL`
		err := r.pool.QueryRow(ctx, q, roleName, featureName).Scan(
			&pc.CanView, &pc.CanCreate, &pc.CanEdit, &pc.CanDelete, &pc.Scope,
		)
		if err != nil {
			return nil, err
		}
		return pc, nil
	}

	// 自治会専用設定を優先し、存在しなければデフォルト設定にフォールバック。
	// ORDER BY (rp.association_id IS NULL) で自治会専用設定（false=0）を先頭に配置し
	// LIMIT 1 で有効な権限を1件取得する。
	const q = `
		SELECT rp.can_view, rp.can_create, rp.can_edit, rp.can_delete, rp.scope
		FROM role_permissions rp
		JOIN roles ro ON ro.id = rp.role_id
		JOIN features f ON f.id = rp.feature_id
		WHERE ro.name = $1 AND f.name = $2
		  AND (rp.association_id = $3 OR rp.association_id IS NULL)
		ORDER BY (rp.association_id IS NULL)
		LIMIT 1`
	err := r.pool.QueryRow(ctx, q, roleName, featureName, *assocID).Scan(
		&pc.CanView, &pc.CanCreate, &pc.CanEdit, &pc.CanDelete, &pc.Scope,
	)
	if err != nil {
		return nil, err
	}
	return pc, nil
}

// UpdatePermissions は権限を一括更新する（system_admin は更新不可）。
// assocID が nil の場合はデフォルト設定を更新する。
// assocID が指定された場合はその自治会専用の設定を更新する。
func (r *Repository) UpdatePermissions(ctx context.Context, operatorID uuid.UUID, assocID *uuid.UUID, inputs []UpdateInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now()
	for _, inp := range inputs {
		// system_admin は変更不可
		var roleName string
		if err := tx.QueryRow(ctx, `SELECT name FROM roles WHERE id = $1`, inp.RoleID).Scan(&roleName); err != nil {
			return fmt.Errorf("get role name: %w", err)
		}
		if roleName == "system_admin" {
			continue
		}

		scope := inp.Scope
		if scope == "" {
			scope = "own_association"
		}

		// 現在値を取得（操作ログ用）
		var beforeJSON []byte
		if assocID == nil {
			_ = tx.QueryRow(ctx, `
				SELECT jsonb_build_object(
					'can_view', can_view, 'can_create', can_create,
					'can_edit', can_edit, 'can_delete', can_delete, 'scope', scope
				) FROM role_permissions WHERE role_id = $1 AND feature_id = $2 AND association_id IS NULL`,
				inp.RoleID, inp.FeatureID,
			).Scan(&beforeJSON)
		} else {
			_ = tx.QueryRow(ctx, `
				SELECT jsonb_build_object(
					'can_view', can_view, 'can_create', can_create,
					'can_edit', can_edit, 'can_delete', can_delete, 'scope', scope
				) FROM role_permissions WHERE role_id = $1 AND feature_id = $2 AND association_id = $3`,
				inp.RoleID, inp.FeatureID, *assocID,
			).Scan(&beforeJSON)
		}

		if assocID == nil {
			// デフォルト設定を UPSERT（部分インデックス role_permissions_default_unique を対象）
			_, err = tx.Exec(ctx, `
				INSERT INTO role_permissions
				    (role_id, feature_id, association_id, can_view, can_create, can_edit, can_delete, scope, updated_at)
				VALUES ($1, $2, NULL, $3, $4, $5, $6, $7, $8)
				ON CONFLICT (role_id, feature_id) WHERE association_id IS NULL DO UPDATE SET
				    can_view   = EXCLUDED.can_view,
				    can_create = EXCLUDED.can_create,
				    can_edit   = EXCLUDED.can_edit,
				    can_delete = EXCLUDED.can_delete,
				    scope      = EXCLUDED.scope,
				    updated_at = EXCLUDED.updated_at`,
				inp.RoleID, inp.FeatureID,
				inp.CanView, inp.CanCreate, inp.CanEdit, inp.CanDelete,
				scope, now,
			)
		} else {
			// 自治会専用設定を UPSERT（部分インデックス role_permissions_assoc_unique を対象）
			_, err = tx.Exec(ctx, `
				INSERT INTO role_permissions
				    (role_id, feature_id, association_id, can_view, can_create, can_edit, can_delete, scope, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				ON CONFLICT (role_id, feature_id, association_id) WHERE association_id IS NOT NULL DO UPDATE SET
				    can_view   = EXCLUDED.can_view,
				    can_create = EXCLUDED.can_create,
				    can_edit   = EXCLUDED.can_edit,
				    can_delete = EXCLUDED.can_delete,
				    scope      = EXCLUDED.scope,
				    updated_at = EXCLUDED.updated_at`,
				inp.RoleID, inp.FeatureID, *assocID,
				inp.CanView, inp.CanCreate, inp.CanEdit, inp.CanDelete,
				scope, now,
			)
		}
		if err != nil {
			return fmt.Errorf("upsert permission: %w", err)
		}

		afterJSON, _ := json.Marshal(map[string]interface{}{
			"can_view": inp.CanView, "can_create": inp.CanCreate,
			"can_edit": inp.CanEdit, "can_delete": inp.CanDelete, "scope": scope,
		})
		targetID := inp.RoleID.String() + ":" + inp.FeatureID.String()
		_, err = tx.Exec(ctx, `
			INSERT INTO operation_logs
			    (operator_id, operation_type, target_type, target_id, before_value, after_value, association_id)
			VALUES ($1, 'permission_update', 'role_permission', $2, $3, $4, $5)`,
			operatorID, targetID, beforeJSON, afterJSON, assocID,
		)
		if err != nil {
			return fmt.Errorf("insert operation log: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// EmergencyAppointment は緊急任命を実行する（ロール変更 + ログ + お知らせ）
func (r *Repository) EmergencyAppointment(ctx context.Context, operatorID uuid.UUID, input EmergencyAppointmentInput) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 対象ユーザーの現在ロールと自治会IDを取得
	var oldRole string
	var assocID *uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT role, association_id FROM users WHERE id = $1 AND is_active = true`, input.UserID).
		Scan(&oldRole, &assocID); err != nil {
		return fmt.Errorf("find user: %w", err)
	}

	// ロール更新
	if _, err := tx.Exec(ctx,
		`UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`,
		input.NewRole, input.UserID,
	); err != nil {
		return fmt.Errorf("update role: %w", err)
	}

	// 操作ログ
	beforeJSON, _ := json.Marshal(map[string]string{"role": oldRole})
	afterJSON, _ := json.Marshal(map[string]string{"role": input.NewRole})
	targetID := input.UserID.String()
	if _, err := tx.Exec(ctx, `
		INSERT INTO operation_logs
		    (operator_id, operation_type, target_type, target_id, before_value, after_value, association_id)
		VALUES ($1, 'emergency_appointment', 'user', $2, $3, $4, $5)`,
		operatorID, targetID, beforeJSON, afterJSON, assocID,
	); err != nil {
		return fmt.Errorf("insert operation log: %w", err)
	}

	// 自治会の全アカウントに通知（assocIDがある場合のみ）
	if assocID != nil {
		noticeBody := fmt.Sprintf("システムからの通知: ユーザーのロールが変更されました。")
		if _, err := tx.Exec(ctx, `
			INSERT INTO notices (id, association_id, title, body, is_pinned, created_by, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, '【緊急任命】ロール変更のお知らせ', $2, false, $3, NOW(), NOW())`,
			*assocID, noticeBody, operatorID,
		); err != nil {
			return fmt.Errorf("create notice: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// GetOperationLogs は操作ログ一覧を取得する（最新50件）
func (r *Repository) GetOperationLogs(ctx context.Context) ([]*OperationLog, error) {
	const q = `
		SELECT id, operator_id, operation_type, target_type, target_id,
		       before_value::text, after_value::text, association_id, created_at
		FROM operation_logs
		ORDER BY created_at DESC
		LIMIT 50`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*OperationLog
	for rows.Next() {
		ol := &OperationLog{}
		if err := rows.Scan(
			&ol.ID, &ol.OperatorID, &ol.OperationType, &ol.TargetType,
			&ol.TargetID, &ol.BeforeValue, &ol.AfterValue, &ol.AssociationID, &ol.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, ol)
	}
	return logs, rows.Err()
}
