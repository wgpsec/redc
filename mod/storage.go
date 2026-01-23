package mod

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	projectBucket = []byte("project")
	caseBucket    = []byte("cases")
)

// Storage handles bbolt database operations
type Storage struct {
	db *bolt.DB
}

// OpenStorage opens or creates a bbolt database for the project
func OpenStorage(projectName string) (*Storage, error) {
	dbPath := filepath.Join(ProjectPath, projectName, "project.db")
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("无法打开数据库: %w", err)
	}

	// Initialize buckets
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(projectBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(caseBucket); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化数据库失败: %w", err)
	}

	return &Storage{db: db}, nil
}

// Close closes the database
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveProject saves project metadata using binary encoding
func (s *Storage) SaveProject(p *RedcProject) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(projectBucket)

		// Convert to proto message
		pbProject := p.ToProto()

		// Encode using gob (Go binary format)
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		if err := enc.Encode(pbProject); err != nil {
			return fmt.Errorf("序列化项目数据失败: %w", err)
		}

		return b.Put([]byte("metadata"), buf.Bytes())
	})
}

// LoadProject loads project metadata only (not cases)
func (s *Storage) LoadProject(projectName string) (*RedcProject, error) {
	var project RedcProject

	err := s.db.View(func(tx *bolt.Tx) error {
		// Load project metadata
		pb := tx.Bucket(projectBucket)
		projectData := pb.Get([]byte("metadata"))
		if projectData == nil {
			return fmt.Errorf("项目元数据不存在")
		}

		var pbProject RedcProjectProto
		buf := bytes.NewBuffer(projectData)
		dec := gob.NewDecoder(buf)
		if err := dec.Decode(&pbProject); err != nil {
			return fmt.Errorf("解析项目数据失败: %w", err)
		}

		project.FromProto(&pbProject)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &project, nil
}

// LoadProjectWithCases loads project metadata and all cases
func (s *Storage) LoadProjectWithCases(projectName string) (*RedcProject, error) {
	var project RedcProject
	var cases []*Case

	err := s.db.View(func(tx *bolt.Tx) error {
		// Load project metadata
		pb := tx.Bucket(projectBucket)
		projectData := pb.Get([]byte("metadata"))
		if projectData == nil {
			return fmt.Errorf("项目元数据不存在")
		}

		var pbProject RedcProjectProto
		buf := bytes.NewBuffer(projectData)
		dec := gob.NewDecoder(buf)
		if err := dec.Decode(&pbProject); err != nil {
			return fmt.Errorf("解析项目数据失败: %w", err)
		}

		project.FromProto(&pbProject)

		// Load all cases
		cb := tx.Bucket(caseBucket)
		c := cb.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var pbCase CaseProto
			caseBuf := bytes.NewBuffer(v)
			caseDec := gob.NewDecoder(caseBuf)
			if err := caseDec.Decode(&pbCase); err != nil {
				return fmt.Errorf("解析 case 数据失败: %w", err)
			}

			caseData := &Case{}
			caseData.FromProto(&pbCase)
			cases = append(cases, caseData)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	project.Case = cases
	return &project, nil
}

// SaveCase saves a single case using binary encoding
func (s *Storage) SaveCase(c *Case) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(caseBucket)

		pbCase := c.ToProto()

		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		if err := enc.Encode(pbCase); err != nil {
			return fmt.Errorf("序列化 case 数据失败: %w", err)
		}

		return b.Put([]byte(c.Id), buf.Bytes())
	})
}

// UpdateCaseState updates only the state fields of a case
func (s *Storage) UpdateCaseState(caseID string, state CaseState, stateTime string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(caseBucket)

		// Read existing case
		data := b.Get([]byte(caseID))
		if data == nil {
			return fmt.Errorf("未找到 case ID: %s", caseID)
		}

		var pbCase CaseProto
		buf := bytes.NewBuffer(data)
		dec := gob.NewDecoder(buf)
		if err := dec.Decode(&pbCase); err != nil {
			return fmt.Errorf("解析 case 数据失败: %w", err)
		}

		// Update state
		pbCase.State = string(state)
		pbCase.StateTime = stateTime

		// Save back
		var outBuf bytes.Buffer
		enc := gob.NewEncoder(&outBuf)
		if err := enc.Encode(&pbCase); err != nil {
			return fmt.Errorf("序列化 case 数据失败: %w", err)
		}

		return b.Put([]byte(caseID), outBuf.Bytes())
	})
}

// DeleteCase removes a case
func (s *Storage) DeleteCase(caseID string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(caseBucket)
		return b.Delete([]byte(caseID))
	})
}

// GetCase retrieves a single case by ID
func (s *Storage) GetCase(caseID string) (*Case, error) {
	var c Case

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(caseBucket)
		data := b.Get([]byte(caseID))
		if data == nil {
			return fmt.Errorf("未找到 case ID: %s", caseID)
		}

		var pbCase CaseProto
		buf := bytes.NewBuffer(data)
		dec := gob.NewDecoder(buf)
		if err := dec.Decode(&pbCase); err != nil {
			return fmt.Errorf("解析 case 数据失败: %w", err)
		}

		c.FromProto(&pbCase)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &c, nil
}

// ListCases returns all cases
func (s *Storage) ListCases() ([]*Case, error) {
	var cases []*Case

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(caseBucket)
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var pbCase CaseProto
			buf := bytes.NewBuffer(v)
			dec := gob.NewDecoder(buf)
			if err := dec.Decode(&pbCase); err != nil {
				return fmt.Errorf("解析 case 数据失败: %w", err)
			}

			caseData := &Case{}
			caseData.FromProto(&pbCase)
			cases = append(cases, caseData)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return cases, nil
}
