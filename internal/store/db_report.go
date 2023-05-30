package store

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/steamid/v2/steamid"
	"go.uber.org/zap"
	"time"
)

type ReportStatus int

const (
	Opened ReportStatus = iota
	NeedMoreInfo
	ClosedWithoutAction
	ClosedWithAction
)

func (status ReportStatus) String() string {
	switch status {
	case ClosedWithoutAction:
		return "Closed without action"
	case ClosedWithAction:
		return "Closed with action"
	case Opened:
		return "Opened"
	default:
		return "Need more information"
	}
}

type Report struct {
	ReportId     int64         `json:"report_id"`
	SourceId     steamid.SID64 `json:"source_id,string"`
	TargetId     steamid.SID64 `json:"target_id,string"`
	Description  string        `json:"description"`
	ReportStatus ReportStatus  `json:"report_status"`
	Reason       Reason        `json:"reason"`
	ReasonText   string        `json:"reason_text"`
	Deleted      bool          `json:"deleted"`
	// Note that we do not use a foreign key here since the demos are not sent until completion
	// and reports can happen mid-game
	DemoName  string    `json:"demo_name"`
	DemoTick  int       `json:"demo_tick"`
	DemoId    int       `json:"demo_id"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

func (report Report) ToURL() string {
	return config.ExtURL("/report/%d", report.ReportId)
}

func NewReport() Report {
	return Report{
		ReportId:     0,
		SourceId:     0,
		Description:  "",
		ReportStatus: 0,
		CreatedOn:    config.Now(),
		UpdatedOn:    config.Now(),
		DemoTick:     -1,
		DemoName:     "",
	}
}

func insertReport(ctx context.Context, report *Report) error {
	const query = `INSERT INTO report (
		    author_id, reported_id, report_status, description, deleted, created_on, updated_on, reason, 
            reason_text, demo_name, demo_tick
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING report_id`
	if errQuery := QueryRow(ctx, query,
		report.SourceId,
		report.TargetId,
		report.ReportStatus,
		report.Description,
		report.Deleted,
		report.CreatedOn,
		report.UpdatedOn,
		report.Reason,
		report.ReasonText,
		report.DemoName,
		report.DemoTick,
	).Scan(&report.ReportId); errQuery != nil {
		return Err(errQuery)
	}
	logger.Info("Report saved",
		zap.Int64("report_id", report.ReportId),
		zap.Int64("author_id", report.SourceId.Int64()))
	return nil
}

func updateReport(ctx context.Context, report *Report) error {
	report.UpdatedOn = config.Now()
	const q = `
		UPDATE report 
		SET author_id = $1, reported_id = $2, report_status = $3, description = $4,
            deleted = $5, updated_on = $6, reason = $7, reason_text = $8, demo_name = $9, demo_tick = $10
        WHERE report_id = $11`
	return Err(Exec(ctx, q, report.SourceId, report.TargetId, report.ReportStatus, report.Description,
		report.Deleted, report.UpdatedOn, report.Reason, report.ReasonText,
		report.DemoName, report.DemoTick, report.ReportId))
}

func SaveReport(ctx context.Context, report *Report) error {
	if report.ReportId > 0 {
		return updateReport(ctx, report)
	}
	return insertReport(ctx, report)
}

func SaveReportMessage(ctx context.Context, message *UserMessage) error {
	if message.MessageId > 0 {
		return updateReportMessage(ctx, message)
	}
	return insertReportMessage(ctx, message)
}

func updateReportMessage(ctx context.Context, message *UserMessage) error {
	message.UpdatedOn = config.Now()
	const query = `
		UPDATE report_message 
		SET deleted = $2, author_id = $3, updated_on = $4, message_md = $5
		WHERE report_message_id = $1
	`
	if errQuery := Exec(ctx, query,
		message.MessageId,
		message.Deleted,
		message.AuthorId,
		message.UpdatedOn,
		message.Message,
	); errQuery != nil {
		return Err(errQuery)
	}
	logger.Info("Report message updated",
		zap.Int64("report_id", message.ParentId),
		zap.Int64("message_id", message.MessageId),
		zap.Int64("author_id", message.AuthorId.Int64()))
	return nil
}

func insertReportMessage(ctx context.Context, message *UserMessage) error {
	const query = `
		INSERT INTO report_message (
		    report_id, author_id, message_md, deleted, created_on, updated_on
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING report_message_id
	`
	if errQuery := QueryRow(ctx, query,
		message.ParentId,
		message.AuthorId,
		message.Message,
		message.Deleted,
		message.CreatedOn,
		message.UpdatedOn,
	).Scan(&message.MessageId); errQuery != nil {
		return Err(errQuery)
	}
	logger.Info("Report message created",
		zap.Int64("report_id", message.ParentId),
		zap.Int64("message_id", message.MessageId),
		zap.Int64("author_id", message.AuthorId.Int64()))
	return nil
}

func DropReport(ctx context.Context, report *Report) error {
	const q = `UPDATE report SET deleted = true WHERE report_id = $1`
	if errExec := Exec(ctx, q, report.ReportId); errExec != nil {
		return Err(errExec)
	}
	logger.Info("Report deleted", zap.Int64("report_id", report.ReportId))
	report.Deleted = true
	return nil
}

func DropReportMessage(ctx context.Context, message *UserMessage) error {
	const q = `UPDATE report_message SET deleted = true WHERE report_message_id = $1`
	if errExec := Exec(ctx, q, message.Message); errExec != nil {
		return Err(errExec)
	}
	logger.Info("Report message deleted", zap.Int64("report_message_id", message.MessageId))
	message.Deleted = true
	return nil
}

type AuthorQueryFilter struct {
	QueryFilter
	AuthorId steamid.SID64 `json:"author_id"`
}

type ReportQueryFilter struct {
	AuthorQueryFilter
	ReportStatus ReportStatus `json:"report_status"`
}

func GetReports(ctx context.Context, opts AuthorQueryFilter) ([]Report, error) {
	var conditions sq.And
	conditions = append(conditions, sq.Eq{"deleted": opts.Deleted})
	if opts.AuthorId > 0 {
		conditions = append(conditions, sq.Eq{"author_id": opts.AuthorId})
	}
	builder := sb.
		Select("r.report_id", "r.author_id", "r.reported_id", "r.report_status",
			"r.description", "r.deleted", "r.created_on", "r.updated_on", "r.reason", "r.reason_text",
			"r.demo_name", "r.demo_tick", "coalesce(d.demo_id, 0)").
		From("report r").
		Where(conditions).
		LeftJoin("demo d on d.title = r.demo_name")

	if opts.Limit > 0 {
		builder = builder.Limit(opts.Limit)
	}
	//if opts.OrderBy != "" {
	//	if opts.SortDesc {
	//		builder = builder.OrderBy(fmt.Sprintf("%s DESC", opts.OrderBy))
	//	} else {
	//		builder = builder.OrderBy(fmt.Sprintf("%s ASC", opts.OrderBy))
	//	}
	//}
	q, a, errSql := builder.ToSql()
	if errSql != nil {
		return nil, Err(errSql)
	}
	rows, errQuery := Query(ctx, q, a...)
	if errQuery != nil {
		return nil, Err(errQuery)
	}
	defer rows.Close()
	var reports []Report
	for rows.Next() {
		var report Report
		if errScan := rows.Scan(
			&report.ReportId,
			&report.SourceId,
			&report.TargetId,
			&report.ReportStatus,
			&report.Description,
			&report.Deleted,
			&report.CreatedOn,
			&report.UpdatedOn,
			&report.Reason,
			&report.ReasonText,
			&report.DemoName,
			&report.DemoTick,
			&report.DemoId,
		); errScan != nil {
			return nil, Err(errScan)
		}
		reports = append(reports, report)
	}
	return reports, nil
}

// GetReportBySteamId returns any open report for the user by the author
func GetReportBySteamId(ctx context.Context, authorId steamid.SID64, steamId steamid.SID64, report *Report) error {
	const query = `
		SELECT 
		   r.report_id, r.author_id, r.reported_id, r.report_status, r.description, 
		   r.deleted, r.created_on, r.updated_on, r.reason, r.reason_text, r.demo_name, r.demo_tick, coalesce(d.demo_id, 0)
		FROM report r
		LEFT JOIN demo d on r.demo_name = d.title
		WHERE deleted = false AND reported_id = $1 AND report_status <= $2 AND author_id = $3`
	if errQuery := QueryRow(ctx, query, steamId, NeedMoreInfo, authorId).
		Scan(
			&report.ReportId,
			&report.SourceId,
			&report.TargetId,
			&report.ReportStatus,
			&report.Description,
			&report.Deleted,
			&report.CreatedOn,
			&report.UpdatedOn,
			&report.Reason,
			&report.ReasonText,
			&report.DemoName,
			&report.DemoTick,
			&report.DemoId,
		); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}
func GetReport(ctx context.Context, reportId int64, report *Report) error {
	const query = `
		SELECT 
		   r.report_id, r.author_id, r.reported_id, r.report_status, r.description, 
		   r.deleted, r.created_on, r.updated_on, r.reason, r.reason_text, r.demo_name, r.demo_tick, 
		   coalesce(d.demo_id, 0)
		FROM report r
		LEFT JOIN demo d on r.demo_name = d.title
		WHERE deleted = false AND report_id = $1`
	if errQuery := QueryRow(ctx, query, reportId).
		Scan(
			&report.ReportId,
			&report.SourceId,
			&report.TargetId,
			&report.ReportStatus,
			&report.Description,
			&report.Deleted,
			&report.CreatedOn,
			&report.UpdatedOn,
			&report.Reason,
			&report.ReasonText,
			&report.DemoName,
			&report.DemoTick,
			&report.DemoId,
		); errQuery != nil {
		return Err(errQuery)
	}
	return nil
}

func GetReportMessages(ctx context.Context, reportId int64) ([]UserMessage, error) {
	const query = `
		SELECT 
		   report_message_id, report_id, author_id, message_md, deleted, created_on, updated_on
		FROM report_message
		WHERE deleted = false AND report_id = $1 
		ORDER BY created_on`
	rows, errQuery := Query(ctx, query, reportId)
	if errQuery != nil {
		if Err(errQuery) == ErrNoResult {
			return nil, nil
		}
	}
	defer rows.Close()
	var messages []UserMessage
	for rows.Next() {
		var msg UserMessage
		if errScan := rows.Scan(
			&msg.MessageId,
			&msg.ParentId,
			&msg.AuthorId,
			&msg.Message,
			&msg.Deleted,
			&msg.CreatedOn,
			&msg.UpdatedOn,
		); errScan != nil {
			return nil, Err(errQuery)
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func GetReportMessageById(ctx context.Context, reportMessageId int64, message *UserMessage) error {
	const query = `
		SELECT 
		   report_message_id, report_id, author_id, message_md, deleted, created_on, updated_on
		FROM report_message
		WHERE report_message_id = $1`
	if errQuery := QueryRow(ctx, query, reportMessageId).
		Scan(
			&message.MessageId,
			&message.ParentId,
			&message.AuthorId,
			&message.Message,
			&message.Deleted,
			&message.CreatedOn,
			&message.UpdatedOn,
		); errQuery != nil {
		return Err(errQuery)
	}

	return nil
}
