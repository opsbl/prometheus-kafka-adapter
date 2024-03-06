package main

import "testing"

func TestRegexNumber(t *testing.T) {
	s := "$1"
	t.Log(RegexGroupByNumber.MathGroupByName(s, "number"))
}

func TestRegexName(t *testing.T) {
	s := "$hostname"
	t.Log(RegexGroupByName.MathGroupByName(s, "name"))
}

func TestLabel(t *testing.T) {
	testLabel := Label{Name: "$node", Value: "$ip"}
	if testLabel.GetLabelType() != LabelIsRegexGroupName {
		t.Fatalf("label type is not group name, type: %v", testLabel.GetLabelType())
	}
	if testLabel.GetValueType() != ValueIsRegexGroupName {
		t.Fatalf("value type is not group name, type: %v", testLabel.GetValueType())
	}
	t.Logf("label name: %s", testLabel.GetName())
	t.Logf("label value: %s", testLabel.GetValue())

	testLabel = Label{Name: "$1", Value: "$2"}
	if testLabel.GetLabelType() != LabelIsRegexGroupIndex {
		t.Fatalf("label type is not group index, type: %v", testLabel.GetLabelType())
	}
	if testLabel.GetValueType() != ValueIsRegxGroupIndex {
		t.Fatalf("value type is not group index, type: %v", testLabel.GetValueType())
	}
	t.Logf("label name: %v", testLabel.GetNameIndex())
	t.Logf("label value: %v", testLabel.GetValueIndex())

	testLabel = Label{Name: "slot", Value: "01"}
	if testLabel.GetLabelType() != LabelIsCustom {
		t.Fatalf("label type is not custom, type: %v", testLabel.GetLabelType())
	}
	if testLabel.GetValueType() != ValueIsCustom {
		t.Fatalf("value type is not custom, type: %v", testLabel.GetValueType())
	}
	t.Logf("label name: %v", testLabel.GetName())
	t.Logf("label value: %v", testLabel.GetValue())

	testLabel = Label{Name: "__name__", Value: "__value__"}
	t.Logf("lable type: %s", testLabel.GetLabelType())
	t.Logf("value type: %v", testLabel.GetValueType())
	t.Logf("label name: %v", testLabel.GetName())
	t.Logf("label value: %v", testLabel.GetValue())

}

func TestRule_Selected(t *testing.T) {
	rule := &Rule{
		Topic: "custom_HOST",
		Selectors: []Selector{
			&MetricNameSelector{
				Method: "regex",
				Value:  "^node_",
			},
		},
		LabelRewriter: nil,
		labelRewriter: nil,
	}
	if err := rule.Init(); err != nil {
		t.Fatalf("init rule failed, %s", err)
	}
	t.Logf("node_cpu_usage, select: %v", rule.Selected("node_cpu_usage"))
	t.Logf("kafka_brokers_number, select: %v", rule.Selected("kafka_brokers_number"))

}

func TestRule_RewriteLabel(t *testing.T) {
	rule := &Rule{
		Topic: "custom_HOST",
		Selectors: []Selector{
			&MetricNameSelector{
				Method: "regex",
				Value:  "^node_",
			},
		},
		LabelRewriter: []*LabelRewriter{
			{
				Name:      "node",
				Overwrite: false,
				Regex:     `(?P<ip>.*?):(?P<port>.*)`, // 10.10.89.61:8080
				Labels: []*Label{
					{Name: "ip", Value: "$1"},
					{Name: "port", Value: "$port"},
				},
				regex: nil,
			},
		},
		labelRewriter: nil,
	}
	if err := rule.Init(); err != nil {
		t.Fatalf("init rule failed, %s", err)
	}
	labels := map[string]string{
		"node": "10.10.89.61:8080",
	}
	rule.RewriteLabel(labels)
	t.Logf("labels: %v", labels)

}
