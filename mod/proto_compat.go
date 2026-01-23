package mod

// Protobuf-compatible structures
// These are simplified protobuf message structures

// RedcProjectProto is the protobuf message for RedcProject
type RedcProjectProto struct {
	ProjectName string `protobuf:"bytes,1,opt,name=project_name,json=projectName,proto3"`
	ProjectPath string `protobuf:"bytes,2,opt,name=project_path,json=projectPath,proto3"`
	CreateTime  string `protobuf:"bytes,3,opt,name=create_time,json=createTime,proto3"`
	User        string `protobuf:"bytes,4,opt,name=user,proto3"`
}

// Reset implements proto.Message
func (m *RedcProjectProto) Reset() { *m = RedcProjectProto{} }

// String implements proto.Message
func (m *RedcProjectProto) String() string { return m.ProjectName }

// ProtoMessage implements proto.Message
func (*RedcProjectProto) ProtoMessage() {}

// CaseProto is the protobuf message for Case
type CaseProto struct {
	Id         string   `protobuf:"bytes,1,opt,name=id,proto3"`
	Name       string   `protobuf:"bytes,2,opt,name=name,proto3"`
	Type       string   `protobuf:"bytes,3,opt,name=type,proto3"`
	Module     string   `protobuf:"bytes,4,opt,name=module,proto3"`
	Operator   string   `protobuf:"bytes,5,opt,name=operator,proto3"`
	Path       string   `protobuf:"bytes,6,opt,name=path,proto3"`
	Node       int32    `protobuf:"varint,7,opt,name=node,proto3"`
	CreateTime string   `protobuf:"bytes,8,opt,name=create_time,json=createTime,proto3"`
	StateTime  string   `protobuf:"bytes,9,opt,name=state_time,json=stateTime,proto3"`
	Parameter  []string `protobuf:"bytes,10,rep,name=parameter,proto3"`
	State      string   `protobuf:"bytes,11,opt,name=state,proto3"`
}

// Reset implements proto.Message
func (m *CaseProto) Reset() { *m = CaseProto{} }

// String implements proto.Message
func (m *CaseProto) String() string { return m.Id }

// ProtoMessage implements proto.Message
func (*CaseProto) ProtoMessage() {}

// ToProto converts RedcProject to protobuf message
func (p *RedcProject) ToProto() *RedcProjectProto {
	return &RedcProjectProto{
		ProjectName: p.ProjectName,
		ProjectPath: p.ProjectPath,
		CreateTime:  p.CreateTime,
		User:        p.User,
	}
}

// FromProto converts protobuf message to RedcProject
func (p *RedcProject) FromProto(pb *RedcProjectProto) {
	p.ProjectName = pb.ProjectName
	p.ProjectPath = pb.ProjectPath
	p.CreateTime = pb.CreateTime
	p.User = pb.User
}

// ToProto converts Case to protobuf message
func (c *Case) ToProto() *CaseProto {
	return &CaseProto{
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
		State:      string(c.State),
	}
}

// FromProto converts protobuf message to Case
func (c *Case) FromProto(pb *CaseProto) {
	c.Id = pb.Id
	c.Name = pb.Name
	c.Type = pb.Type
	c.Module = pb.Module
	c.Operator = pb.Operator
	c.Path = pb.Path
	c.Node = int(pb.Node)
	c.CreateTime = pb.CreateTime
	c.StateTime = pb.StateTime
	c.Parameter = pb.Parameter
	c.State = CaseState(pb.State)
}
