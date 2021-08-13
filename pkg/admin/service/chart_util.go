package service

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type VASPRunXml struct {
	// XMLName xml.Name    `xml:"modeling"`
	InCAR      []Option    `xml:"incar>i"`
	Structures []Structure `xml:"structure"`
	DoS        DOS         `xml:"calculation>dos"`
	Eigen      EIGEN       `xml:"calculation>eigenvalues>array"`
	KPoints    []VArray    `xml:"kpoints>varray"`
	Generated  VArray      `xml:"kpoints>generation"`
	Atoms      struct {
		Count int `xml:"atoms"`
		Types int `xml:"types"`
		Info  []struct {
			Name string `xml:"name,attr"`
			Data []struct {
				Atom []string `xml:"c"`
			} `xml:"set>rc"`
		} `xml:"array"`
	} `xml:"atominfo"`
	Steps []Steps `xml:"calculation>scstep"`
}


type EIGEN struct {
	Dimenstions []Option `xml:"dimension"`
	Fields      []string `xml:"field"`
	Set         struct {
		Comment string `xml:"comment,attr"`
		Spin    []struct {
			Data []Set `xml:"set"`
		} `xml:"set"`
	} `xml:"set"`
}

type DOS struct {
	Efermi Option `xml:"i"`
	Total  struct {
		Dimenstions []Option `xml:"dimension"`
		Fields      []string `xml:"field"`
		Data        []Set    `xml:"set>set"`
	} `xml:"total>array"`
	Partial struct {
		Dimenstions []Option `xml:"dimension"`
		Fields      []string `xml:"field"`
		Set         struct {
			Comment string `xml:"comment,attr"`
			Ions    []struct {
				Comment string `xml:"comment,attr"`
				Spins   []Set  `xml:"set"`
			} `xml:"set"`
		} `xml:"set"`
	} `xml:"partial>array"`
}

type Set struct {
	Comment string   `xml:"comment,attr"`
	Data    []string `xml:"r"`
}

type Option struct {
	Dim   string `xml:"dim,attr"`
	Name  string `xml:"name,attr"`
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type VArray struct {
	Name     string   `xml:"name,attr"`
	Param    string   `xml:"param,attr"`
	Data     []Option `xml:"v"`
	Division Option   `xml:"i"`
}

type Structure struct {
	Name    string `xml:"name,attr"`
	Crystal struct {
		List   []VArray `xml:"varray"`
		Volume Option   `xml:"i"`
	} `xml:"crystal"`
	Position VArray `xml:"varray"`
}

type Steps struct {
	Time []struct {
		Name  string `xml:"name,attr"`
		Value string `xml:",chardata"`
	}
	Energy []struct {
		Name  string `xml:"name,attr"`
		Value string `xml:",chardata"`
	} `xml:"energy>i"`
}

var encoding = regexp.MustCompile(`(?i)encoding="[^"]+"`)
var utfEncoding = regexp.MustCompile(`(?i)encoding="utf-\d"`)
var empty = " "
var spaces = regexp.MustCompile(`[ \t]+`)
var marks = regexp.MustCompile(`((?:[ \t]+-?\d+\.\d+){3})[ \t]+(?:\![ \t]+)?(\w+)`)

var fsteps = regexp.MustCompile(` \d+ F= [^=]+E0= ([^ ]+)[^\n]+`)
var davE = regexp.MustCompile(`DAV: +\d+ +([^ ]+)`)

func parse(flpath string) (VASPRunXml, error) {
	var v VASPRunXml
	if _, err := os.Stat(flpath); os.IsNotExist(err) {
		return v, err
	}
	// fmt.Println(flpath)
	body, err := ioutil.ReadFile(flpath)
	if err != nil {
		return v, err
	}

	i := bytes.Index(body, []byte("<modeling>"))
	if i < 0 {
		return v, fmt.Errorf("Invalid vasprun-test.xml")
	}

	found := encoding.Find(body[:i])
	if found != nil && !utfEncoding.Match(found) {
		body = bytes.Replace(body, found, []byte(`encoding="UTF-8"`), 1)
	}
	err = xml.Unmarshal(body, &v)
	return v, err
}

func getPoscarFromXml(flpath string) (buff bytes.Buffer, err error) {
	v, err := parse(flpath)
	if err != nil {
		return buff, err
	}
	counts := map[string]int{}
	keys := []string{}
	for _, info := range v.Atoms.Info {
		if info.Name == "atoms" {
			for _, ainfo := range info.Data {
				name := strings.Trim(ainfo.Atom[0], " ")
				if name == "r" { // TODO, a vasprun-test.xml bug?
					name = "Zr"
				} else if name == "X" {
					name = "Xe"
				}
				if n, ok := counts[name]; ok {
					counts[name] = n + 1
				} else {
					counts[name] = 1
					keys = append(keys, name)
				}
				// atoms = append(atoms, name)
			}
		}
	}
	buff = bytes.Buffer{}
	for _, k := range keys {
		num := counts[k]
		buff.WriteString(fmt.Sprintf("%s%d", k, num))
	}
	buff.Write([]byte("\n1.0\n"))

	s := v.Structures[len(v.Structures) - 1]
	for _, item := range s.Crystal.List {
		if item.Name != "basis" {
			continue
		}
		for _, opt := range item.Data {
			sl := spaces.Split(strings.Trim(opt.Value, empty), -1)
			if len(sl) >= 3 {
				v0, _ := strconv.ParseFloat(sl[0], 10)
				v1, _ := strconv.ParseFloat(sl[1], 10)
				v2, _ := strconv.ParseFloat(sl[2], 10)
				buff.WriteString(fmt.Sprintf("%12.6f %12.6f %12.6f\n", v0, v1, v2))
			}
		}
	}
	for _, k := range keys {
		buff.WriteString(fmt.Sprintf("%4s", k))
	}
	buff.Write([]byte("\n"))
	for _, k := range keys {
		num := counts[k]
		buff.WriteString(fmt.Sprintf("%4d", num))
	}
	buff.Write([]byte("\nDirect\n"))
	for _, opt := range s.Position.Data {
		sl := spaces.Split(strings.Trim(opt.Value, empty), -1)
		if len(sl) >= 3 {
			v0, _ := strconv.ParseFloat(sl[0], 10)
			v1, _ := strconv.ParseFloat(sl[1], 10)
			v2, _ := strconv.ParseFloat(sl[2], 10)
			buff.WriteString(fmt.Sprintf("%12.6f %12.6f %12.6f\n", v0, v1, v2))
		}
	}
	return buff, nil
}

func getDosFromXml(flpath string) (buff bytes.Buffer, err error) {
	v, err := parse(flpath)
	if err != nil {
		return
	}

	efermi, _ := strconv.ParseFloat(strings.Trim(v.DoS.Efermi.Value, empty), 10)

	x := []float64{}
	spin := [][]float64{}
	xmin, xmax := float64(0), float64(0)
	for i, ds := range v.DoS.Total.Data {
		yl := []float64{}
		for _, n := range ds.Data {
			sl := spaces.Split(strings.Trim(n, empty), -1)
			if i == 0 {
				v0, _ := strconv.ParseFloat(sl[0], 10)
				if v0 < xmin {
					xmin = v0
				} else if v0 > xmax {
					xmax = v0
				}
				x = append(x, v0)
			}
			v1, _ := strconv.ParseFloat(sl[1], 10)
			yl = append(yl, v1)
		}
		spin = append(spin, yl)
	}

	partialIon := []interface{}{}
	partialX := []float64{}
	fields := []string{}
	names := []string{}
	if len(v.DoS.Partial.Fields) > 1 {
		names = v.DoS.Partial.Fields[1:]
	}
	for i, fld := range names {
		fld = strings.Trim(fld, empty)
		names[i] = fld
		if fld == "x2-y2" {
			fld = "dx2-y2"
		}
		fields = append(fields, fld)
	}
	gotX := false
	for _, ion := range v.DoS.Partial.Set.Ions {
		partialSpin := [][][]float64{}
		for _, ds := range ion.Spins {
			yl := make([][]float64, len(fields))
			for _, n := range ds.Data {
				sl := spaces.Split(strings.Trim(n, empty), -1)
				if !gotX {
					vf, _ := strconv.ParseFloat(sl[0], 10)
					partialX = append(partialX, vf)
				}
				for j, _ := range fields {
					vf, _ := strconv.ParseFloat(sl[1+j], 10)
					yl[j] = append(yl[j], vf)
				}
			}
			gotX = true
			partialSpin = append(partialSpin, yl)
		}
		partialIon = append(partialIon, partialSpin)
	}

	atoms := []string{}
	for _, info := range v.Atoms.Info {
		if info.Name == "atoms" {
			for _, ainfo := range info.Data {
				name := strings.Trim(ainfo.Atom[0], " ")
				if name == "r" {
					name = "Zr"
				} else if name == "X" {
					name = "Xe"
				}
				atoms = append(atoms, name)
			}
		}
	}

	body, _ := json.Marshal(map[string]interface{}{
		"total": map[string]interface{}{
			"x":    x,
			"spin": spin,
		},
		"partial": map[string]interface{}{
			"fields": fields,
			"x":      partialX,
			"ion":    partialIon,
			"atoms":  atoms,
		},
		"efermi": efermi,
	})
	buff.Write(body)
	return buff, nil
}

func getEigenFromXml(flpath string) (buff bytes.Buffer, err error) {
	v, err := parse(flpath)
	if err != nil {
		return
	}
	x := []float64{}
	efermi, _ := strconv.ParseFloat(strings.Trim(v.DoS.Efermi.Value, empty), 10)

	for _, kp := range v.KPoints {
		if kp.Name != "kpointlist" {
			continue
		}
		last := []float64{0, 0, 0}
		dl := float64(0)
		for _, kd := range kp.Data {
			sl := spaces.Split(kd.Value, -1)
			if len(sl) < 3 {
				continue
			}
			vx, _ := strconv.ParseFloat(sl[0], 10)
			vy, _ := strconv.ParseFloat(sl[1], 10)
			vz, _ := strconv.ParseFloat(sl[2], 10)

			dist := math.Sqrt((vx-last[0])*(vx-last[0]) +
				(vy-last[1])*(vy-last[1]) + (vz-last[2])*(vz-last[2]))
			if len(x) > 0 && dist == 0 {
				continue
			}
			dl += dist
			x = append(x, dl)
			last = []float64{vx, vy, vz}
		}
	} // x

	tags := []string{}
	generated := []float64{}
	kpn := []int{}
	if v.Generated.Param == "listgenerated" || v.Generated.Param == "Gamma" {
		kpfl := filepath.Join(filepath.Dir(flpath), "KPOINTS")
		if _, err := os.Stat(kpfl); !os.IsNotExist(err) {
			// read marks from KPOINTS
			body, err := ioutil.ReadFile(kpfl)
			if err == nil && len(body) > 0 {
				m := marks.FindAllSubmatch(body, -1)
				if len(m) > 0 {
					last := []byte("")
					for _, e := range m {
						if bytes.Equal(last, e[1]) {
							continue
						}
						last = e[1]
						tags = append(tags, string(e[2]))
					}
				}
			}
		}

		gv := [][]float64{}
		kpv := []string{}
		if len(v.Generated.Division.Value) > 0 {
			kpv = spaces.Split(strings.Trim(v.Generated.Division.Value, empty), -1)
		} else if len(v.Generated.Data) > 0 && v.Generated.Data[0].Name == "divisions" {
			kpv = spaces.Split(strings.Trim(v.Generated.Data[0].Value, empty), -1)
			v.Generated.Data = v.Generated.Data[1:]
		}
		for i := 0; i < len(kpv); i++ {
			n, _ := strconv.ParseInt(kpv[i], 10, 64)
			kpn = append(kpn, int(n) - 1)
		}

		for _, kd := range v.Generated.Data {
			sl := spaces.Split(strings.Trim(kd.Value, empty), -1)
			if len(sl) < 3 {
				continue
			}
			vx, _ := strconv.ParseFloat(sl[0], 10)
			vy, _ := strconv.ParseFloat(sl[1], 10)
			vz, _ := strconv.ParseFloat(sl[2], 10)
			gv = append(gv, []float64{vx, vy, vz})
		}
		if len(kpn) > 0 {
			basis := [][]float64{}
			s := v.Structures[len(v.Structures) - 1]
			for _, item := range s.Crystal.List {
				if item.Name != "rec_basis" {
					continue
				}
				for _, opt := range item.Data {
					sl := spaces.Split(strings.Trim(opt.Value, empty), -1)
					if len(sl) >= 3 {
						v0, _ := strconv.ParseFloat(sl[0], 10)
						v1, _ := strconv.ParseFloat(sl[1], 10)
						v2, _ := strconv.ParseFloat(sl[2], 10)
						basis = append(basis, []float64{v0, v1, v2})
					}
				}
			}

			dxyz := [][]float64{}
			for _, v := range gv {
				m := []float64{0, 0, 0}
				for j := 0; j < 3; j++ {
					m[0] += v[j] * basis[j][0]
					m[1] += v[j] * basis[j][1]
					m[2] += v[j] * basis[j][2]
				}
				dxyz = append(dxyz, m)
			}

			dl := []float64{}
			for i := 0; i < len(dxyz)-1; i++ {
				dl = append(dl, math.Sqrt((dxyz[i+1][0]-dxyz[i][0])*(dxyz[i+1][0]-dxyz[i][0])+
					(dxyz[i+1][1]-dxyz[i][1])*(dxyz[i+1][1]-dxyz[i][1])+
					(dxyz[i+1][2]-dxyz[i][2])*(dxyz[i+1][2]-dxyz[i][2])))
			}
			w := float64(0)
			generated = []float64{0}
			x = []float64{}
			for i := 0; i < len(dl); i++ {
				x = append(x, linspace(w, w+dl[i], kpn[0])...)
				w += dl[i]
				generated = append(generated, w)
			}
			// fmt.Println(x, generated, dl)
		}
	}

	spin := [][][]float64{}
	if len(kpn) > 0 {
		for _, dss := range v.Eigen.Set.Spin {
			d := [][]float64{}
			for _, ds := range dss.Data {
				for i, n := range ds.Data {
					sl := spaces.Split(strings.Trim(n, empty), -1)
					if len(sl) > 0 {
						v0, _ := strconv.ParseFloat(sl[0], 10)
						if len(d) < i+1 {
							d = append(d, []float64{})
						}
						d[i] = append(d[i], v0)
					}
				}
			}
			spin = append(spin, d)
		}
	}
	body, _ := json.Marshal(map[string]interface{}{
		"x":      x,
		"spin":   spin,
		"gx":     generated,
		"tags":   tags,
		"efermi": efermi,
	})

	buff.Write(body)
	return buff, nil
}

func getEnergyFromXml(flpath, key string) (buff bytes.Buffer, err error) {
	dtype, energy, err := steps(flpath, key)
	if err != nil {
		return buff, err
	}
	body, _ := json.Marshal(map[string]interface{}{
		"type":   dtype,
		"energy": energy,
	})
	buff.Write(body)
	return buff, nil
}

func steps(flpath, key string) (dtype string, energy string, err error) {
	dtype = "f"
	v, err := parse(flpath)
	energies := []string{}

	for _, step := range v.Steps {
		for _, e := range step.Energy {
			if e.Name == key {
				energies = append(energies, strings.Trim(e.Value, empty))
			}
		}
	}
	energy = strings.Join(energies, ",")
	return
}

func linspace(start, stop float64, sample int) (result []float64) {
	step := (stop - start) / float64(sample-1)
	for i := 0; i < sample; i++ {
		result = append(result, start+float64(i)*step)
	}
	return
}