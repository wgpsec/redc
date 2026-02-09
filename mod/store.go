package mod

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	// 【请修改这里】根据你的 go.mod 替换为正确的 pb 路径
	// 例如你的 module 叫 "red-cloud"，这里就是 "red-cloud/pb"
	"red-cloud/pb"

	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

const (
	// DBPath 数据库文件路径
	DBPath = "redc.db"
	// BucketProjectMeta 存放项目元信息的桶名称
	BucketProjectMeta = "Project_Meta"
)

// ==========================================
// 0. 基础数据库连接封装 (带超时锁)
// ==========================================

// dbExec 负责打开数据库、执行事务、关闭数据库
// 解决了 CLI 工具多进程同时运行时的文件锁问题
func dbExec(fn func(tx *bolt.Tx) error) error {
	// 设置 2 秒超时：如果另一个 redc 进程正在写，当前进程会等待，而不是报错退出
	opts := &bolt.Options{Timeout: 2 * time.Second}

	// 打开数据库 (0600 权限只允许当前用户读写)
	db, err := bolt.Open(filepath.Join(RedcPath, DBPath), 0600, opts)
	if err != nil {
		if err == bolt.ErrTimeout {
			return fmt.Errorf("数据库正忙(被锁定)，请稍后重试")
		}
		return fmt.Errorf("无法打开数据库: %v", err)
	}
	defer db.Close() // 确保函数结束时释放锁

	return db.Update(fn)
}

// ==========================================
// 1. 转换逻辑 (Mapper) - 极简字符串版
// ==========================================

// toProto: 将业务对象 Case 转为存储对象 pb.Case
func (c *Case) toProto() *pb.Case {
	// 1. 处理私有字段 output (map类型)
	// Protobuf 不支持复杂 map，我们把它序列化成 JSON 字符串存进去
	outputBytes, _ := json.Marshal(c.output)

	return &pb.Case{
		Id:         c.Id,
		Name:       c.Name,
		Type:       c.Type,
		Module:     c.Module,
		Operator:   c.Operator,
		Path:       c.Path,
		Node:       int32(c.Node),
		CreateTime: c.CreateTime,
		StateTime:  c.StateTime,
		Parameter:  c.Parameter,

		// 【简化】直接强转 string，不再使用 Enum 映射
		State: c.State,

		// 存储序列化后的 Map
		OutputJson: string(outputBytes),
	}
}

// caseFromProto: 将存储对象 pb.Case 转回业务对象 Case
func caseFromProto(p *pb.Case) *Case {
	c := &Case{
		Id:         p.Id,
		Name:       p.Name,
		Type:       p.Type,
		Module:     p.Module,
		Operator:   p.Operator,
		Path:       p.Path,
		Node:       int(p.Node),
		CreateTime: p.CreateTime,
		StateTime:  p.StateTime,
		Parameter:  p.Parameter,
		State:      p.State,
	}

	// 还原 output map
	if len(p.OutputJson) > 0 {
		// 直接操作私有字段 output，因为我们在同一个 package mod 下
		json.Unmarshal([]byte(p.OutputJson), &c.output)
	}

	return c
}

// ==========================================
// 2. Case 存取操作 (Active Record 风格)
// ==========================================

// DBSave 将当前 Case 保存到数据库 (原子操作)
// 逻辑：Case -> Proto -> Bytes -> BoltDB
func (c *Case) DBSave() error {
	if c.ProjectID == "" {
		return fmt.Errorf("严重错误: Case %s 丢失了 ProjectID，无法保存", c.Id)
	}
	return dbExec(func(tx *bolt.Tx) error {
		// 每个项目有独立的 Bucket，例如 "Cases_default"
		bucketName := []byte(fmt.Sprintf("Cases_%s", c.ProjectID))

		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return err
		}

		// 序列化
		pbData := c.toProto()
		data, err := proto.Marshal(pbData)
		if err != nil {
			return fmt.Errorf("序列化失败: %v", err)
		}

		// 写入 (Key=CaseID)
		return b.Put([]byte(c.Id), data)
	})
}

// DBRemove 从数据库中删除当前 Case
func (c *Case) DBRemove() error {
	if c.ProjectID == "" {
		return fmt.Errorf("严重错误: Case %s 丢失了 ProjectID，无法删除", c.Id)
	}
	return dbExec(func(tx *bolt.Tx) error {
		bucketName := []byte(fmt.Sprintf("Cases_%s", c.ProjectID))
		b := tx.Bucket(bucketName)
		if b == nil {
			return nil // 桶不存在，也就是数据本来就没有，视为成功
		}
		return b.Delete([]byte(c.Id))
	})
}

// LoadProjectCases 加载指定项目下的所有 Case
func LoadProjectCases(projectName string) ([]*Case, error) {
	var cases []*Case

	err := dbExec(func(tx *bolt.Tx) error {
		bucketName := []byte(fmt.Sprintf("Cases_%s", projectName))
		b := tx.Bucket(bucketName)
		if b == nil {
			return nil // 没数据，返回空切片
		}

		// 遍历桶内所有数据
		return b.ForEach(func(k, v []byte) error {
			var p pb.Case
			// 反序列化 Proto
			if err := proto.Unmarshal(v, &p); err == nil {
				// 转为业务对象
				cases = append(cases, caseFromProto(&p))
			}
			return nil
		})
	})

	// 按创建时间降序排序（最新的在前面）
	sort.Slice(cases, func(i, j int) bool {
		return cases[i].CreateTime > cases[j].CreateTime
	})

	return cases, err
}

// ==========================================
// 3. Project 元数据存取操作
// ==========================================

// SaveMeta 保存项目元数据 (注意：不保存 Case 列表)
func (p *RedcProject) SaveMeta() error {
	return dbExec(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte(BucketProjectMeta))

		// 构造 Proto 对象 (仅元数据)
		pbProj := &pb.Project{
			ProjectName: p.ProjectName,
			ProjectPath: p.ProjectPath,
			CreateTime:  p.CreateTime,
			User:        p.User,
		}

		data, err := proto.Marshal(pbProj)
		if err != nil {
			return err
		}

		return b.Put([]byte(p.ProjectName), data)
	})
}

// LoadProjectMeta 读取项目元数据
func LoadProjectMeta(name string) (*RedcProject, error) {
	var p pb.Project

	err := dbExec(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketProjectMeta))
		if b == nil {
			return fmt.Errorf("无项目数据")
		}

		data := b.Get([]byte(name))
		if data == nil {
			return fmt.Errorf("项目不存在: %s", name)
		}

		return proto.Unmarshal(data, &p)
	})

	if err != nil {
		return nil, err
	}

	// 转回业务对象
	return &RedcProject{
		ProjectName: p.ProjectName,
		ProjectPath: p.ProjectPath,
		CreateTime:  p.CreateTime,
		User:        p.User,
	}, nil
}

// ListAllProjects 获取数据库中所有项目列表
func ListAllProjects() ([]*RedcProject, error) {
	var projects []*RedcProject

	err := dbExec(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BucketProjectMeta))
		if b == nil {
			return nil // 桶不存在，返回空列表
		}

		// 遍历桶内所有项目
		return b.ForEach(func(k, v []byte) error {
			var p pb.Project
			if err := proto.Unmarshal(v, &p); err == nil {
				// 过滤掉特殊的结果目录（不应该作为项目显示）
				if p.ProjectName == "redc-taskresult" || p.ProjectName == "task-result" {
					return nil
				}
				projects = append(projects, &RedcProject{
					ProjectName: p.ProjectName,
					ProjectPath: p.ProjectPath,
					CreateTime:  p.CreateTime,
					User:        p.User,
				})
			}
			return nil
		})
	})

	return projects, err
}
