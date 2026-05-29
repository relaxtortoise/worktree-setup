package config

import "strings"

// Config 表示完整配置（任意一层）
type Config struct {
	MainWorktree string        `yaml:"main_worktree,omitempty"`
	PathStrategy *PathStrategy `yaml:"path_strategy,omitempty"`
	On           *Events       `yaml:"on,omitempty"`
}

// PathStrategy 可以是字符串或模板对象
type PathStrategy struct {
	Name     string `yaml:"-"`
	Template string `yaml:"-"`
}

func (p PathStrategy) MarshalYAML() (any, error) {
	if p.Template != "" {
		return map[string]string{"template": p.Template}, nil
	}
	return p.Name, nil
}

func (p *PathStrategy) UnmarshalYAML(unmarshal func(any) error) error {
	var name string
	if err := unmarshal(&name); err == nil {
		p.Name = name
		return nil
	}
	var obj struct {
		Template string `yaml:"template"`
	}
	if err := unmarshal(&obj); err != nil {
		return err
	}
	p.Template = obj.Template
	return nil
}

type Events struct {
	PreCreate    *Event `yaml:"pre-create,omitempty"`
	PostCreate   *Event `yaml:"post-create,omitempty"`
	PostCheckout *Event `yaml:"post-checkout,omitempty"`
	PreDelete    *Event `yaml:"pre-delete,omitempty"`
	PostDelete   *Event `yaml:"post-delete,omitempty"`
}

type Step struct {
	Run     string     `yaml:"run,omitempty"`
	Copy    *CopyItems `yaml:"copy,omitempty"`
	Symlink *CopyItems `yaml:"symlink,omitempty"`
}

func (s *Step) UnmarshalYAML(unmarshal func(any) error) error {
	// 尝试裸字符串 → 隐式 run
	var str string
	if err := unmarshal(&str); err == nil {
		s.Run = str
		return nil
	}
	// 尝试对象 {run:, copy:, symlink:}
	type stepAlias Step
	var alias stepAlias
	if err := unmarshal(&alias); err != nil {
		return err
	}
	*s = Step(alias)
	return nil
}

type Event struct {
	Steps   []Step     `yaml:"steps,omitempty"`
	Run     []string   `yaml:"run,omitempty"`
	Copy    *CopyItems `yaml:"copy,omitempty"`
	Symlink *CopyItems `yaml:"symlink,omitempty"`
}

// StepsOrLegacy 统一返回 steps 列表。若 events 用三段式则转为 virtual steps。
func (e *Event) StepsOrLegacy() []Step {
	if e.Steps != nil {
		return e.Steps
	}
	var steps []Step
	if e.Copy != nil && len(e.Copy.Items) > 0 {
		steps = append(steps, Step{Copy: e.Copy})
	}
	if e.Symlink != nil && len(e.Symlink.Items) > 0 {
		steps = append(steps, Step{Symlink: e.Symlink})
	}
	if len(e.Run) > 0 {
		steps = append(steps, Step{Run: strings.Join(e.Run, "\n")})
	}
	return steps
}

// CopyAction 单个复制/软链接条目
type CopyAction struct {
	From string
	To   string
}

// CopyItems 支持 map 或 list 两种 YAML 形式，解析时统一为 []CopyAction
type CopyItems struct {
	Items []CopyAction
}

func (c *CopyItems) UnmarshalYAML(unmarshal func(any) error) error {
	// 尝试 map 形式
	var m map[string]string
	if err := unmarshal(&m); err == nil {
		for from, to := range m {
			if to == "" {
				to = from
			}
			c.Items = append(c.Items, CopyAction{From: from, To: to})
		}
		return nil
	}

	// 尝试 list 形式
	var raw []any
	if err := unmarshal(&raw); err != nil {
		return err
	}
	for _, item := range raw {
		switch v := item.(type) {
		case string:
			from, to := parseColonShorthand(v)
			c.Items = append(c.Items, CopyAction{From: from, To: to})
		case map[string]any:
			from, _ := v["from"].(string)
			to, _ := v["to"].(string)
			c.Items = append(c.Items, CopyAction{From: from, To: to})
		}
	}
	return nil
}

func parseColonShorthand(s string) (string, string) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return s, s
}
