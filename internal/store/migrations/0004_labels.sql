CREATE TABLE IF NOT EXISTS labels (
    id          TEXT PRIMARY KEY,
    company_id  TEXT NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    color       TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    UNIQUE(company_id, name)
);

CREATE TABLE IF NOT EXISTS issue_labels (
    issue_id    TEXT NOT NULL REFERENCES issues(id) ON DELETE CASCADE,
    label_id    TEXT NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    created_at  TEXT NOT NULL,
    PRIMARY KEY (issue_id, label_id)
);

CREATE INDEX IF NOT EXISTS issue_labels_label_idx   ON issue_labels(label_id);
CREATE INDEX IF NOT EXISTS labels_company_idx       ON labels(company_id);
