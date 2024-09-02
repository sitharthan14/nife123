package ruleengine

import (
	"encoding/json"
	"strings"

	"github.com/nifetency/nife.io/pkg/ruleengine/constants"
)

const (
	OperatorAnd = "and"
	OperatorOr  = "or"
	NoOperator  = ""
)

var defaultComparators = map[string]Comparator{
	"eq":        equal,
	"neq":       notEqual,
	"gt":        greaterThan,
	"gte":       greaterThanEqual,
	"lt":        lessThan,
	"lte":       lessThanEqual,
	"contains":  contains,
	"ncontains": notContains,
	"oneof":     oneOf,
	"noneof":    noneOf,
	"regex":     regex,
}

type condition struct {
	Comparator string      `json:"comparator"`
	Parameter  string      `json:"parameter"`
	Value      interface{} `json:"value"`
}

type ruleResponse struct {
	Response []string `json:"response"`
}

type appRuleThen struct {
	RuleResponse ruleResponse `json:"ruleResponse"`
}

type appRuleIf struct {
	Operator    string      `json:"operator"`
	Conditions  []condition `json:"conditions"`
	AppRuleThen appRuleThen `json:"then"`
}

type appRuleElseIf struct {
	Operator    string      `json:"operator"`
	Conditions  []condition `json:"conditions"`
	AppRuleThen appRuleThen `json:"then"`
}

type appRuleElse struct {
	AppRuleThen appRuleThen `json:"then"`
}

type rule struct {
	AppRuleIf     appRuleIf       `json:"if"`
	AppRuleElseIf []appRuleElseIf `json:"elseif,omitempty"`
	AppRuleElse   *appRuleElse    `json:"else,omitempty"`
}

type appRule struct {
	Rule rule `json:"rule"`
}

type application struct {
	Application string    `json:"application"`
	AppRules    []appRule `json:"appRules"`
}

type Engine struct {
	Application application `json:"application"`
	comparators map[string]Comparator
}

func copyRuleResponseValue(response1 ruleResponse, response2 ruleResponse) ruleResponse {
	// if response2.Response != nil {
	// 	response1.Response = append(response1.Response, response2.Response[0])
	// }

	if response2.Response != nil {
		response1.Response = response2.Response
	}

	return response1
}

func (r *condition) MarshalJSON() ([]byte, error) {
	type unmappedRule struct {
		Comparator string      `json:"comparator"`
		Parameter  string      `json:"parameter"`
		Value      interface{} `json:"value"`
	}

	switch t := r.Value.(type) {
	case map[interface{}]struct{}:
		var s []interface{}
		for k := range t {
			s = append(s, k)
		}
		r.Value = s
	}

	umr := unmappedRule{
		Comparator: r.Comparator,
		Parameter:  r.Parameter,
		Value:      r.Value,
	}

	return json.Marshal(umr)
}

func (r *condition) UnmarshalJSON(data []byte) error {
	type mapRule struct {
		Comparator string      `json:"comparator"`
		Parameter  string      `json:"parameter"`
		Value      interface{} `json:"value"`
	}

	var mr mapRule
	err := json.Unmarshal(data, &mr)
	if err != nil {
		return err
	}

	switch t := mr.Value.(type) {
	case []interface{}:
		var m = make(map[interface{}]struct{})
		for _, v := range t {
			m[v] = struct{}{}
		}

		mr.Value = m
	}

	*r = condition{
		Comparator: mr.Comparator,
		Parameter:  mr.Parameter,
		Value:      mr.Value,
	}

	return nil
}

func NewJSONEngine(raw json.RawMessage) (Engine, error) {
	var e Engine
	err := json.Unmarshal(raw, &e)
	if err != nil {
		return Engine{}, err
	}
	e.comparators = defaultComparators
	return e, nil
}

func (e Engine) AddComparator(name string, c Comparator) Engine {
	e.comparators[name] = c
	return e
}

// Evaluate will ensure all of the composites in the engine are true
func (e Engine) Evaluate(props map[string]interface{}) (bool, *ruleResponse) {
	finalRuleResponse := ruleResponse{}
	c := e.Application
	val := GetKeyValue(props, constants.AppKayName)
	if val != nil {
		if val == c.Application {
			for _, a := range c.AppRules {
				appRuleThen := a.evaluate(props, e.comparators)
				if appRuleThen != nil {
					finalRuleResponse = copyRuleResponseValue(finalRuleResponse, appRuleThen.RuleResponse)
					println(strings.Join(finalRuleResponse.Response, " "))
				}
			}
		}
	}
	return true, &finalRuleResponse
}

func (ar appRule) evaluate(props map[string]interface{}, comps map[string]Comparator) *appRuleThen {
	res := false
	rule := ar.Rule
	res = rule.AppRuleIf.evaluate(props, comps)
	if res == true {
		return &rule.AppRuleIf.AppRuleThen
	} else {
		if rule.AppRuleElseIf != nil {
			for _, elseIfCondition := range rule.AppRuleElseIf {
				res = elseIfCondition.evaluate(props, comps)
				if res == true {
					return &elseIfCondition.AppRuleThen
				}
			}
		}
		if res != true {
			if rule.AppRuleElse != nil {
				return &rule.AppRuleElse.AppRuleThen
			}
		}
	}
	return nil
}

// Evaluate will return true if the rule is true, false otherwise
func (r condition) evaluate(props map[string]interface{}, comps map[string]Comparator) bool {
	// Make sure we can get a value from the props
	val := pluck(props, r.Parameter)
	if val == nil {
		return false
	}

	comp, ok := comps[r.Comparator]
	if !ok {
		return false
	}

	return comp(val, r.Value)
}

func (c appRuleIf) evaluate(props map[string]interface{}, comps map[string]Comparator) bool {
	switch c.Operator {
	case OperatorAnd:
		for _, r := range c.Conditions {
			val := pluck(props, r.Parameter)
			if val == nil {
				continue
			}
			res := r.evaluate(props, comps)
			if res == false {
				return false
			}
		}
		return true
	case NoOperator:
		for _, r := range c.Conditions {
			val := pluck(props, r.Parameter)
			if val == nil {
				continue
			}
			res := r.evaluate(props, comps)
			if res == false {
				return false
			}
		}
		return true
	case OperatorOr:
		for _, r := range c.Conditions {
			res := r.evaluate(props, comps)
			if res == true {
				return true
			}
		}
		return false
	}

	return false
}

func (c appRuleElseIf) evaluate(props map[string]interface{}, comps map[string]Comparator) bool {
	switch c.Operator {
	case OperatorAnd:
		for _, r := range c.Conditions {
			val := pluck(props, r.Parameter)
			if val == nil {
				continue
			}
			res := r.evaluate(props, comps)
			if res == false {
				return false
			}
		}
		return true
	case NoOperator:
		for _, r := range c.Conditions {
			val := pluck(props, r.Parameter)
			if val == nil {
				continue
			}
			res := r.evaluate(props, comps)
			if res == false {
				return false
			}
		}
		return true
	case OperatorOr:
		for _, r := range c.Conditions {
			res := r.evaluate(props, comps)
			if res == true {
				return true
			}
		}
		return false
	}

	return false
}
