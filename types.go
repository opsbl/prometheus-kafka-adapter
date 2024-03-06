package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	RegexGroupByNumber = &RegexHandler{regexp.MustCompile(`\$(?P<number>\d+$)`)}
	RegexGroupByName   = &RegexHandler{regexp.MustCompile(`\$(?P<name>[^0-9]\w+)`)}
)

const (
	SelectEQ        = "eq"
	SelectStartWith = "start_with"
	SelectRegex     = "regex"
)

type LabelType int
type ValueType int

const (
	LabelIsLabel LabelType = iota
	LabelIsRegexGroupName
	LabelIsRegexGroupIndex
	LabelIsCustom
)

const (
	ValueIsValue ValueType = iota
	ValueIsRegexGroupName
	ValueIsRegxGroupIndex
	ValueIsCustom
)

var (
	LabelTypes = []string{
		"Label",
		"RegexGroupName",
		"RegexGroupIndex",
		"Custom",
	}
	ValueTypes = []string{
		"Value",
		"RegexGroupName",
		"RegexGroupIndex",
		"Custom",
	}
)

func (l LabelType) String() string {
	if l < LabelIsLabel || l > LabelIsCustom {
		return "Unknown"
	}
	return LabelTypes[l]
}

func (l ValueType) String() string {
	if l < ValueIsValue || l > ValueIsCustom {
		return "Unknown"
	}
	return ValueTypes[l]
}

// Selector 选择器接口
type Selector interface {
	Init() error
	Match(string) bool
	String() string
}

type RegexHandler struct {
	*regexp.Regexp
}

func (r *RegexHandler) MathGroupByName(str, name string) (bool, string) {
	res := r.FindStringSubmatch(str)
	return r.StrResultGroupByName(res, name)
}

func (r *RegexHandler) StrResultGroupByName(res []string, name string) (bool, string) {
	if res != nil && len(res) > 0 {
		for i, n := range r.SubexpNames() {
			if n == name {
				return true, res[i]
			}
		}
	}
	return false, ""
}

// MetricNameSelector 指标名称选择器
type MetricNameSelector struct {
	Method string `yaml:"method" validate:"required,oneof=eq start_with regex"`
	Value  string `yaml:"value" validate:"required"`
	regex  *regexp.Regexp
}

func (s *MetricNameSelector) String() string {
	return fmt.Sprintf("MetricNameSelector(method=%s,value=%s)", s.Method, s.Value)
}

func (s *MetricNameSelector) Init() error {

	switch s.Method {
	case SelectRegex:
		regex, err := regexp.Compile(s.Value)
		if err != nil {
			return err
		}
		s.regex = regex
	}

	return nil
}

// Match 匹配指标名称是否满足既定规则
func (s *MetricNameSelector) Match(name string) bool {
	switch s.Method {
	case SelectEQ:
		if name == s.Value {
			return true
		}
	case SelectStartWith:
		if strings.HasPrefix(name, s.Value) {
			return true
		}
	case SelectRegex:
		if s.regex.MatchString(name) {
			return true
		}
	}

	return false
}

type Label struct {
	Name  string `yaml:"name" validate:"required"`
	Value string `yaml:"value" validate:"required"`

	nameAsStr   string
	nameAsIndex int

	valueAsStr   string
	valueAsIndex int

	_name        bool
	_value       bool
	_reloadName  bool
	_reloadValue bool

	nameType  LabelType
	valueType ValueType
}

func (l *Label) GetValueType() ValueType {

	if !l._reloadValue && l._value {
		return l.valueType
	}

	l._value = true
	l._reloadValue = false
	if l.Value == "__value__" {
		l.valueType = ValueIsValue
		return l.valueType
	}
	if strings.HasPrefix(l.Value, "$") {
		ok, number := RegexGroupByNumber.MathGroupByName(l.Value, "number")
		if ok {
			value, _ := strconv.ParseInt(number, 10, 64)
			l.valueAsIndex = int(value)
			l.valueType = ValueIsRegxGroupIndex
			return l.valueType
		}
		ok, name := RegexGroupByName.MathGroupByName(l.Value, "name")
		if ok {
			l.valueAsStr = name
			l.valueType = ValueIsRegexGroupName
			return l.valueType
		}
	}
	l.valueType = ValueIsCustom
	l.valueAsStr = l.Value
	return l.valueType
}

func (l *Label) GetLabelType() LabelType {

	if !l._reloadName && l._name {
		return l.nameType
	}
	l._reloadName = false
	l._name = true
	if l.Name == "__name__" {
		l.nameType = LabelIsLabel
		return l.nameType
	}

	if strings.HasPrefix(l.Name, "$") {
		ok, number := RegexGroupByNumber.MathGroupByName(l.Name, "number")
		if ok {
			value, _ := strconv.ParseInt(number, 10, 64)
			l.nameAsIndex = int(value)
			l.nameType = LabelIsRegexGroupIndex
			return l.nameType
		}
		ok, name := RegexGroupByName.MathGroupByName(l.Name, "name")
		if ok {
			l.nameAsStr = name
			l.nameType = LabelIsRegexGroupName
			return l.nameType
		}
	}
	l.nameAsStr = l.Name
	l.nameType = LabelIsCustom
	return l.nameType
}

func (l *Label) GetName() string {
	if !l._name {
		l.GetLabelType()
	}
	return l.nameAsStr
}

func (l *Label) GetNameIndex() int {
	if !l._name {
		l.GetLabelType()
	}
	if l.nameType == LabelIsRegexGroupIndex {
		return l.nameAsIndex
	}
	return -1
}

func (l *Label) GetValue() string {
	if !l._value {
		l.GetValueType()
	}
	return l.valueAsStr
}

func (l *Label) GetValueIndex() int {
	if !l._name {
		l.GetValueType()
	}
	if l.valueType == ValueIsRegxGroupIndex {
		return l.valueAsIndex
	}
	return -1
}

func (l *Label) Reload() {
	l._reloadName = true
	l._reloadValue = true
}

type LabelRewriter struct {
	Name      string   `yaml:"name" validate:"required"`
	Overwrite bool     `yaml:"overwrite"`
	Regex     string   `yaml:"regex" validate:"required"`
	Labels    []*Label `yaml:"labels" validate:"required"`
	regex     *RegexHandler
}

func (l *LabelRewriter) Init() error {
	if regex, err := regexp.Compile(l.Regex); err != nil {
		return err
	} else {
		l.regex = &RegexHandler{regex}
	}
	return nil
}

// GenNewLabels 生成新的标签
func (l *LabelRewriter) GenNewLabels(key, value string) map[string]string {
	if key != l.Name {
		return map[string]string{}
	}
	if len(l.Labels) == 0 {
		return map[string]string{}
	}
	match := l.regex.FindStringSubmatch(value)
	if len(match) == 0 {
		return make(map[string]string)
	}

	labels := make(map[string]string, 1)
	for _, label := range l.Labels {

		if label.Name == "" {
			continue
		}
		_name := ""
		_value := ""
		switch label.GetLabelType() {
		case LabelIsLabel:
			_name = key
		case LabelIsRegexGroupName:
			_, _name = l.regex.StrResultGroupByName(match, label.GetName())
		case LabelIsRegexGroupIndex:
			idx := label.GetNameIndex()
			if idx > -1 && idx < len(match) {
				_name = match[idx]
			}
		case LabelIsCustom:
			_name = label.GetName()
		}

		switch label.GetValueType() {
		case ValueIsValue:
			_value = value
		case ValueIsRegexGroupName:
			_, _value = l.regex.StrResultGroupByName(match, label.GetValue())
		case ValueIsRegxGroupIndex:
			idx := label.GetValueIndex()
			if idx > -1 && idx < len(match) {
				_value = match[idx]
			}
		case ValueIsCustom:
			_value = label.GetValue()
		}
		labels[_name] = _value
	}

	return labels

}

type Rule struct {
	Topic         string               `yaml:"topic" validate:"required"`
	Selectors     []MetricNameSelector `yaml:"selectors" validate:"required"`
	LabelRewriter []*LabelRewriter     `yaml:"labelRewriter"`
	Org           int                  `yaml:"org"`
	Token         string               `yaml:"token" validate:"required"`
	DeleteLabels  map[string]bool      `yaml:"deleteLabels"` // 指定删除标签
	labelRewriter map[string]*LabelRewriter
}

func (r *Rule) Selected(name string) bool {
	for _, selector := range r.Selectors {
		if selector.Match(name) {
			return true
		}
	}
	return false
}

func (r *Rule) RewriteLabel(labels map[string]string) map[string]string {
	for key, value := range labels {
		rewriter, ok := r.labelRewriter[key]
		if !ok {
			continue
		}
		newLabels := rewriter.GenNewLabels(key, value)
		if len(newLabels) > 0 && rewriter.Overwrite {
			delete(labels, key)
		}
		for _key, _value := range newLabels {
			labels[_key] = _value
		}

	}
	return labels
}

func (r *Rule) Init() error {
	r.labelRewriter = make(map[string]*LabelRewriter, 1)
	if r.LabelRewriter != nil {
		for idx, rewriter := range r.LabelRewriter {
			if err := rewriter.Init(); err != nil {
				return fmt.Errorf("rewriter(%d:name=%s) init failed, error: %s", idx, rewriter.Name, err)
			}
			r.labelRewriter[rewriter.Name] = rewriter
		}
	}
	if r.Selectors != nil {
		for idx, selector := range r.Selectors {
			if err := selector.Init(); err != nil {
				return fmt.Errorf("rewriter(%d:method=%s) init failed, error: %s", idx, selector, err)
			}
		}
	}

	return nil
}

type EasyOpsConfig struct {
	DefaultORG int     `yaml:"defaultOrg" validate:"required"`
	Rules      []*Rule `yaml:"rules"`
}

func (e *EasyOpsConfig) Init() (err error) {
	for _, rule := range e.Rules {
		if err = rule.Init(); err != nil {
			return
		}
	}
	return nil
}

func (e *EasyOpsConfig) Select(name string) (bool, *Rule) {
	if e.Rules == nil {
		return false, nil
	}
	for _, rule := range e.Rules {
		if rule.Selected(name) {
			return true, rule
		}
	}
	return false, nil
}
