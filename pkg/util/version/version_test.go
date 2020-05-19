package version

// Copyright (c) Microsoft Corporation.
// Licensed under the Apache License 2.0.

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

func TestNewVersion(t *testing.T) {
	for i, tt := range []struct {
		vs   []byte
		want *Version
	}{
		{
			vs:   []byte{1, 2},
			want: &Version{V: [3]byte{1, 2}},
		},
		{
			want: &Version{},
		},
		{
			vs:   []byte{1, 2, 3, 4},
			want: &Version{V: [3]byte{1, 2, 3}},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := NewVersion(tt.vs...)
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(got)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	for _, tt := range []struct {
		vsn     string
		want    *Version
		wantErr string
	}{
		{
			vsn:  "4.3.0-0.nightly-2020-04-17-062811",
			want: &Version{V: [3]byte{4, 3}, Suffix: "-0.nightly-2020-04-17-062811"},
		},
		{
			vsn:  "40.30.10",
			want: &Version{V: [3]byte{40, 30, 10}},
		},
		{
			vsn:     "bad",
			wantErr: `could not parse version "bad"`,
		},
	} {
		t.Run(tt.vsn, func(t *testing.T) {
			got, err := ParseVersion(tt.vsn)
			if err != nil && err.Error() != tt.wantErr ||
				err == nil && tt.wantErr != "" {
				t.Error(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(got)
			}
		})
	}
}

func TestLt(t *testing.T) {
	for i, tt := range []struct {
		input *Version
		min   *Version
		want  bool
	}{
		{
			input: NewVersion(4, 1),
			min:   NewVersion(4, 3),
			want:  true,
		},
		{
			input: NewVersion(4, 4),
			min:   NewVersion(4, 3, 1),
		},
		{
			input: NewVersion(4, 4),
			min:   NewVersion(4, 4),
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got := tt.input.Lt(tt.min)
			if got != tt.want {
				t.Error(got)
			}
		})
	}
}

func TestGetUpgrade(t *testing.T) {
	Upgrade43 := Upgrade{
		Version: NewVersion(4, 3, 18),
	}
	Upgrade44 := Upgrade{
		Version: NewVersion(4, 4, 3),
	}

	Upgrades = append([]Upgrade{}, Upgrade43, Upgrade44)

	for i, tt := range []struct {
		input *Version
		want  *Upgrade
		err   error
	}{
		{
			input: NewVersion(4, 3, 1),
			want:  &Upgrade43,
			err:   nil,
		},
		{
			input: NewVersion(4, 4),
			want:  &Upgrade44,
			err:   nil,
		},
		{
			input: NewVersion(4, 5),
			err:   fmt.Errorf("upgrade for %d not found", NewVersion(4, 5, 0).V[:]),
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			upgrade, err := GetUpgrade(tt.input)
			if tt.err != err && !reflect.DeepEqual(err, tt.err) {
				t.Error(err)
			}
			if upgrade != nil && !reflect.DeepEqual(upgrade, tt.want) {
				t.Error(fmt.Sprintf("\n%+v\n !=\n%+v", upgrade, tt.want))
			}
		})
	}
}

func TestVersionString(t *testing.T) {
	for i, tt := range []struct {
		input *Version
		want  string
	}{
		{
			input: NewVersion(4, 3, 1),
			want:  "4.3.1",
		},
		{
			input: NewVersion(4, 3),
			want:  "4.3.0",
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if tt.want != tt.input.String() {
				t.Error(fmt.Sprintf("%s!=%s", tt.want, tt.input.String()))
			}
		})
	}

}
