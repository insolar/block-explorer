//
// Copyright 2019 Insolar Technologies GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package insconfig

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const placeholder = "<-key->"

// Params for config parsing
type Params struct {
	// EnvPrefix is a prefix for environment variables
	EnvPrefix string
	// ViperHooks is custom viper decoding hooks
	ViperHooks []mapstructure.DecodeHookFunc
	// ConfigPathGetter should return config path
	ConfigPathGetter ConfigPathGetter
	// FileNotRequired - do not return error on file not found
	FileNotRequired bool
}

// ConfigPathGetter - implement this if you don't want to use config path from --config flag
type ConfigPathGetter interface {
	GetConfigPath() string
}

type insConfigurator struct {
	params Params
	viper  *viper.Viper
}

// New creates new insConfigurator with params
func New(params Params) insConfigurator {
	return insConfigurator{
		params: params,
		viper:  viper.New(),
	}
}

// Load loads configuration from path, env and makes checks
// configStruct is a pointer to your config
func (i *insConfigurator) Load(configStruct interface{}) error {
	if i.params.EnvPrefix == "" {
		return errors.New("EnvPrefix should be defined")
	}
	if i.params.ConfigPathGetter == nil {
		return errors.New("ConfigPathGetter should be defined")
	}

	configPath := i.params.ConfigPathGetter.GetConfigPath()
	return i.load(configPath, configStruct)
}

func (i *insConfigurator) load(path string, configStruct interface{}) error {

	i.viper.AutomaticEnv()
	i.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	i.viper.SetEnvPrefix(i.params.EnvPrefix)

	i.viper.SetConfigFile(path)
	if err := i.viper.ReadInConfig(); err != nil {
		if !i.params.FileNotRequired {
			return err
		}
		fmt.Printf("failed to load config from '%s'\n", path)
	}

	// this 'if' block necessary for check duplicated map keys in YAML
	if !i.params.FileNotRequired {
		bytes, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to read config file")
		}
		err = yaml.UnmarshalStrict(bytes, configStruct)
		if err != nil && strings.Contains(err.Error(), "already set in map") {
			return errors.Wrapf(err, "failed to unmarshal config file into configuration structure")
		}
	}

	i.params.ViperHooks = append(i.params.ViperHooks, mapstructure.StringToTimeDurationHookFunc(), mapstructure.StringToSliceHookFunc(","))
	err := i.viper.UnmarshalExact(configStruct, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		i.params.ViperHooks...,
	)))
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal config file into configuration structure")
	}
	configStructKeys, err := deepFieldNames(configStruct, "", false)
	if err != nil {
		return err
	}

	configStructKeys, mapKeys := separateKeys(configStructKeys)
	configStructKeys, err = i.checkNoExtraENVValues(configStructKeys, mapKeys)
	if err != nil {
		return err
	}

	for k := range mapKeys {
		if used := mapKeys[k]; !used {
			configStructKeys = append(configStructKeys, k)
		}
	}

	err = i.checkAllValuesIsSet(configStructKeys)
	if err != nil {
		return err
	}

	// Second Unmarshal needed because of bug https://github.com/spf13/viper/issues/761
	// This should be evaluated after manual values overriding is done
	err = i.viper.UnmarshalExact(configStruct, viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		i.params.ViperHooks...,
	)))
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal config file into configuration structure 2")
	}
	return nil
}

func (i *insConfigurator) checkNoExtraENVValues(structKeys []string, mapKeys map[string]bool) ([]string, error) {
	var errorKeys []string
	prefixLen := len(i.params.EnvPrefix)
	for _, e := range os.Environ() {
		if len(e) > prefixLen && e[0:prefixLen]+"_" == strings.ToUpper(i.params.EnvPrefix)+"_" {
			kv := strings.SplitN(e, "=", 2)
			key := strings.ReplaceAll(strings.Replace(strings.ToLower(kv[0]), i.params.EnvPrefix+"_", "", 1), "_", ".")

			if k, pref, match := matchMapKey(mapKeys, key); match && !stringInSlice(key, structKeys) {
				structKeys = append(structKeys, newKeys(mapKeys, k, pref)...)
			}

			if stringInSlice(key, structKeys) {
				// This manually sets value from ENV and overrides everything, this temporarily fix issue https://github.com/spf13/viper/issues/761
				i.viper.Set(key, kv[1])
			} else {
				errorKeys = append(errorKeys, key)
			}
		}
	}
	if len(errorKeys) > 0 {
		return structKeys, errors.New(fmt.Sprintf("Wrong config keys found in ENV: %s", strings.Join(errorKeys, ", ")))
	}
	return structKeys, nil
}

func separateKeys(list []string) ([]string, map[string]bool) {
	var structKeys []string
	mapKeys := make(map[string]bool)
	for _, s := range list {
		if strings.Contains(s, placeholder) {
			mapKeys[s] = false
		} else {
			structKeys = append(structKeys, s)
		}
	}
	return structKeys, mapKeys
}

func newKeys(keys map[string]bool, key, pref string) []string {
	var names []string
	oldStr := strings.Join([]string{pref, placeholder}, "")
	newStr := strings.Join([]string{pref, key}, "")
	for k := range keys {
		if strings.HasPrefix(k, oldStr) {
			names = append(names, strings.Replace(k, oldStr, newStr, 1))
			keys[k] = true
		}
	}
	return names
}

func matchMapKey(keys map[string]bool, key string) (string, string, bool) {
	for k := range keys {
		l := strings.ToLower(k)
		pattern := strings.ReplaceAll(l, ".", "\\.")
		pattern = strings.ReplaceAll(pattern, placeholder, ".+")
		match, err := regexp.MatchString(pattern, key)
		if err != nil {
			fmt.Println(err)
		}
		if match {
			parts := strings.Split(l, placeholder)
			return strings.TrimSuffix(strings.TrimPrefix(key, parts[0]), parts[1]), parts[0], true
		}
	}
	return "", "", false
}

func (i *insConfigurator) checkAllValuesIsSet(structKeys []string) error {
	var errorKeys []string
	allKeys := i.viper.AllKeys()
	for _, keyName := range structKeys {
		if !i.viper.IsSet(keyName) {
			// Due to a bug https://github.com/spf13/viper/issues/447 we can't use InConfig, so
			if !stringInSlice(keyName, allKeys) {
				errorKeys = append(errorKeys, keyName)
			}
			// Value of this key is "null" but it's set in config file
		}
	}
	if len(errorKeys) > 0 {
		return errors.New(fmt.Sprintf("Keys is not defined in config: %s", strings.Join(errorKeys, ", ")))
	}
	return nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if strings.ToLower(b) == strings.ToLower(a) {
			return true
		}
	}
	return false
}

func deepFieldNames(iface interface{}, prefix string, inMap bool) ([]string, error) {
	names := make([]string, 0)
	ifv := reflect.Indirect(reflect.ValueOf(iface))

	switch ifv.Kind() {
	case reflect.Struct:
		for i := 0; i < ifv.Type().NumField(); i++ {
			v := ifv.Field(i)
			tagValue := ifv.Type().Field(i).Tag.Get("mapstructure")
			tagParts := strings.Split(tagValue, ",")

			// If "squash" is specified in the tag, we squash the field down.
			squash := false
			for _, tag := range tagParts[1:] {
				if tag == "squash" {
					squash = true
					break
				}
			}

			newPrefix := ""
			currPrefix := ""
			if !squash {
				currPrefix = ifv.Type().Field(i).Name
			}
			if prefix != "" {
				newPrefix = strings.Join([]string{prefix, currPrefix}, ".")
			} else {
				newPrefix = currPrefix
			}

			fieldNames, err := deepFieldNames(v.Interface(), strings.ToLower(newPrefix), inMap)
			if err != nil {
				return nil, err
			}
			names = append(names, fieldNames...)
		}
	case reflect.Map:
		if inMap {
			return nil, errors.New("nested maps are not allowed in config")
		}
		inMap = true
		keyKind := ifv.Type().Key().Kind()
		if keyKind != reflect.String {
			return nil, errors.New(fmt.Sprintf("maps in config must have string keys but got: %s key in %s", keyKind, ifv.Type()))
		}

		if len(ifv.MapKeys()) != 0 {
			for _, k := range ifv.MapKeys() {
				key := k.String()
				newPrefix := ""
				if prefix != "" {
					newPrefix = strings.Join([]string{prefix, key}, ".")
				} else {
					newPrefix = key
				}

				fieldNames, err := deepFieldNames(ifv.MapIndex(k).Interface(), strings.ToLower(newPrefix), inMap)
				if err != nil {
					return nil, err
				}
				names = append(names, fieldNames...)
			}
		} else {
			newPrefix := ""
			if prefix != "" {
				newPrefix = strings.Join([]string{prefix, placeholder}, ".")
			} else {
				newPrefix = placeholder
			}

			e := ifv.Type().Elem()
			value := reflect.Zero(e)
			fieldNames, err := deepFieldNames(value.Interface(), strings.ToLower(newPrefix), inMap)
			if err != nil {
				return nil, err
			}
			names = append(names, fieldNames...)
		}
		inMap = false
	default:
		if prefix != "" {
			names = append(names, strings.ToLower(prefix))
		}
	}

	return names, nil
}

// ToYaml returns yaml marshalled struct
func (i *insConfigurator) ToYaml(c interface{}) string {
	// todo clean password
	out, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("failed to marshal config structure: %v", err)
	}
	return string(out)
}

// Empty config generation part

// you may use insconfig:"default|comment" Tag on struct fields to express your feelings.

type YamlTemplatable interface {
	TemplateTo(w io.Writer, m *YamlTemplater) error
}

type YamlTemplater struct {
	Obj   interface{}       // what are we marshaling right now
	Level int               // Level of recursion
	Tag   reflect.StructTag // Tag for current field
	FName string            // current field name
}

func NewYamlTemplater(obj interface{}) *YamlTemplater {
	return &YamlTemplater{
		Obj:   obj,
		Level: -1,
		Tag:   "",
	}
}

func (m *YamlTemplater) TemplateTo(w io.Writer) error {

	if o, ok := m.Obj.(YamlTemplatable); ok {
		return o.TemplateTo(w, m)
	}

	// HINT  SOLVE me need the same to work on (z *Z)TemplateTo()

	t := reflect.TypeOf(m.Obj)
	v := reflect.ValueOf(m.Obj)

	if t.Kind() == reflect.Ptr {
		m.Obj = v.Elem().Interface()
		return m.TemplateTo(w)
	}

	indent := ""
	if m.Level > 0 {
		indent = strings.Repeat("  ", m.Level)
	}
	d := ""
	c := ""
	yfname := ""
	if cont, ok := m.Tag.Lookup("insconfig"); ok { // detect tags
		arr := strings.SplitN(cont, "|", 2)
		d = arr[0]
		c = arr[1]
	}
	if cont, ok := m.Tag.Lookup("yaml"); ok {
		yfname = cont
	}

	if c != "" { //write down a comment
		if _, err := fmt.Fprintf(w, "%s#%s\n", indent, c); err != nil {
			return err
		}
	}
	if m.FName != "" && t.Kind() != reflect.Array {
		if yfname == "" {
			yfname = strings.ToLower(m.FName)
		}
		if _, err := fmt.Fprintf(w, "%s%s: ", indent, yfname); err != nil {
			return err
		}
	}

	switch t.Kind() { // main switch
	case reflect.Struct: //no default
		if _, err := fmt.Fprint(w, "\n"); err != nil {
			return err
		}
		for i := 0; i < t.NumField(); i++ {
			if err := (&YamlTemplater{
				Obj:   v.Field(i).Interface(),
				Level: m.Level + 1,
				Tag:   t.Field(i).Tag,
				FName: t.Field(i).Name,
			}).TemplateTo(w); err != nil {
				return errors.Wrapf(err, "in field %s", t.Field(i).Name)
			}
		}

	case reflect.Map:
		_, err := fmt.Fprintf(w, "# <map> of %s \n", t.Elem().Name())
		return err

	case reflect.Array, reflect.Slice:
		_, err := fmt.Fprintf(w, "# <array> of %s \n", t.Elem().Name())
		return err

	case reflect.String, // all scalars
		reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16,
		reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128:
		_, err := fmt.Fprintf(w, "%s # %s\n", d, t.Name())
		return err

	default:
		return fmt.Errorf("unknown serialization for type %s kind %s (please implement YamlTemplatable)", t.Name(), t.Kind())
	}
	return nil
}
