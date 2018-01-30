package config

import (
    "fmt"
    "bufio"
    "io"
    "os"
    "strings"
)

type ConfLineError struct {
    line string
    error string
}

type ConfFileError struct {
    fname string
    error *ConfLineError
}

type NoSuchSectionError struct {
    sec_name string
}

type NoSuchKeyError struct {
    sec_name string
    key string
}

func (cle *ConfLineError) Error() string {
    return fmt.Sprintf("%s: %s", cle.line, cle.error)
}

func (cfe *ConfFileError) Error() string {
    return fmt.Sprintf("configuration file error: %s\n\t%s", cfe.fname, cfe.error)
}

func (nsse *NoSuchSectionError) Error() string {
    return fmt.Sprintf("no such section [%s] was set", nsse.sec_name)
}

func (nske *NoSuchKeyError) Error() string {
    return fmt.Sprintf("no such key [%s] was set under section [%s]", nske.key, nske.sec_name)
}

func console_log(level, message string) {
    fmt.Printf("[%s] %s\n", level, message)
}

/* ini配置格式的节, 形如[default]
 * name: 节名
 * fields: 节下的配置项及其值
 */
type Section struct {
    name string
    fields map[string]string
}

type PState struct {
    in_sec bool
    cur_sec *Section
}

/* 配置句柄
 * conf_files: 存放配置文件路径及对应的已打开文件句柄
 * sections: 存放已解析的节
 */
type Config struct {
    conf_files map[string]*os.File
    sections map[string]*Section
    _pstate *PState
}

/* Section构造函数
 */
func NewSection(name string) *Section {
    var _section Section
    _section.name = name
    _section.fields = make(map[string]string)
    return &_section
}

/* Config构造函数
 */
func NewConfig(files []string) *Config {
    var _config Config
    _config.conf_files = make(map[string]*os.File)
    for _, file := range files {
        _config.conf_files[file] = nil
    }
    _config.sections = make(map[string]*Section)
    _config._pstate = &PState{false, nil}
    return &_config
}

/* 打开配置文件
 */
func (config *Config) _open_files() (bool, error) {
    for fpath, fobj := range config.conf_files {
        if fobj == nil {
            fobj, err := os.Open(fpath)
            if err != nil {
                console_log("ERROR", err.Error())
                return false, err
            } else {
                config.conf_files[fpath] = fobj
            }
        }
    }
    return true, nil
}

/* 解析单个行
 */
func (config *Config) _parse_line(line string) (bool, *ConfLineError) {
    if len(line) > 0 {
        line = strings.TrimSpace(line)
        line_len := len(line)
        if line_len > 0 {
            if (line[0] == '[' && line[line_len - 1] == ']') {
                sec_name := line[1: line_len - 1]
                if len(sec_name) == 0 {
                    return false, &ConfLineError{line, "section name cannot be empty"}
                }
                pSection, _in := config.sections[sec_name]
                if !_in {
                    pSection = NewSection(sec_name)
                    config.sections[sec_name] = pSection
                }
                config._pstate.in_sec = true
                config._pstate.cur_sec = pSection
            } else {
                first_equation_pos := strings.Index(line, "=")
                if first_equation_pos <= 0 {
                    return false, &ConfLineError{line, "invalid configuration line"}
                }
                key := line[: first_equation_pos]
                key = strings.TrimSpace(key)
                if len(key) == 0 {
                    return false, &ConfLineError{line, "key cannot be empty"}
                }
                value := line[first_equation_pos + 1:]
                value = strings.TrimSpace(value)
                if value[0] == '"' && value[len(value) - 1] == '"'{
                    value = value[1: len(value) - 1]
                }
                if !config._pstate.in_sec {
                    return false, &ConfLineError{line, "configuration line without section"}
                }
                cur_sec := config._pstate.cur_sec
                // we just cover the value set before
                cur_sec.fields[key] = value
            }
        }
    }
    
    return true, nil
}

/* 解析单个配置文件
 */
func (config *Config) _parse_file(fname string, file *os.File) (bool, error) {
    if file != nil {
        br := bufio.NewReader(file)
        //in_section := false
        //var cur_section *Section = nil
        for {
            line, _, err := br.ReadLine()
            if err == nil || err == io.EOF {
                _success, _err := config._parse_line(string(line))
                if !_success {
                    return false, &ConfFileError{fname, _err}
                }
            }
            if err == io.EOF {
                break
            } else if err != nil {
                console_log("ERROR", err.Error())
                return false, err
            }
        }
    }
    return true, nil
}

/* 解析完成后的善后工作
 */
func (config *Config) finalize() {
    for _, fobj := range config.conf_files {
        fobj.Close()
    }
}

/* 解析配置
 */
func (config *Config) Parse_conf() (bool, error) {
    _success, _err := config._open_files()
    defer config.finalize()
    if !_success {
        return false, _err
    }
    for fname, fobj := range config.conf_files {
        _success, _err = config._parse_file(fname, fobj)
        if !_success {
            return false, _err
        }
    }
    return true, nil
}

/* 获取指定节下的指定关键字的值
 */
func (config *Config) Get(section, key string) (string, error) {
    pSection, _in := config.sections[section]
    if !_in {
        return "", &NoSuchSectionError{section}
    }
    value, _in := pSection.fields[key]
    if !_in {
        return "", &NoSuchKeyError{section, key}
    }
    return value, nil
}

