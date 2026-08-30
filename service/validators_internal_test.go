package service

import (
	"errors"
	"testing"
)

// The field-level half of the load-time guards. Like the registry-level
// checks in validate_internal_test.go, none of these can be reached
// through Load: the shipped data is valid, so only an internal test can
// hand them something malformed.
//
// They matter more than their shape suggests. This layer is what stops
// wrong rule data loading silently, and a broken guard here would not
// surface as a crash or a failed test — it would surface as a character
// generated against a rule the book does not contain, which nothing finds
// until someone re-reads the page.

// rejects runs a table of malformed inputs and asserts each is refused
// with ErrInvalidData. The one shared helper for every validator below:
// they differ only in what they are handed.
func rejects[T any](t *testing.T, validate func(T) error, cases map[string]T) {
	t.Helper()

	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			err := validate(input)
			if err == nil {
				t.Fatal("want an error, got nil")
			}

			if !errors.Is(err, ErrInvalidData) {
				t.Errorf("want ErrInvalidData, got %v", err)
			}
		})
	}
}

// valid is a service that passes every guard, so a case can break exactly
// one thing and know that is what the error is about.
func valid() *Service {
	svc := &Service{
		Name:        "Navy",
		DraftNumber: 1,
		Enlistment:  ThrowSpec{Target: "8+"},
		Survival:    ThrowSpec{Target: "5+"},
		Reenlist:    ThrowSpec{Target: "6+"},
		Commission:  &ThrowSpec{Target: "10+"},
		Promotion:   &ThrowSpec{Target: "8+"},
		Ranks:       []string{"Ensign"},
		Muster: Muster{
			Benefits: make([]Benefit, 7),
			Cash:     []int{1, 2, 3, 4, 5, 6, 7},
		},
	}
	rows := make([]SkillResult, 6)

	for i := range rows {
		rows[i] = SkillResult{Skill: "Gambling"}
	}

	svc.Skills = SkillTables{
		PersonalDevelopment: rows,
		ServiceSkills:       rows,
		AdvancedEducation:   rows,
		AdvancedEducation8:  rows,
	}

	return svc
}

// broken applies one mutation to an otherwise valid service.
func broken(mutate func(*Service)) *Service {
	svc := valid()
	mutate(svc)

	return svc
}

func TestValidateServiceRejectsBadData(t *testing.T) {
	// The baseline must pass, or every case below proves nothing.
	if err := validateService(valid()); err != nil {
		t.Fatalf("the valid baseline does not validate: %v", err)
	}

	rejects(t, validateService, map[string]*Service{
		"a name outside the book's six":  broken(func(s *Service) { s.Name = "Pirates" }),
		"draft number below 1":           broken(func(s *Service) { s.DraftNumber = 0 }),
		"draft number above 6":           broken(func(s *Service) { s.DraftNumber = 7 }),
		"an unparseable throw target":    broken(func(s *Service) { s.Survival.Target = "banana" }),
		"a promotion path with no ranks": broken(func(s *Service) { s.Ranks = nil }),
		"a muster table short a row":     broken(func(s *Service) { s.Muster.Benefits = make([]Benefit, 6) }),
		"a skills table short a row": broken(func(s *Service) {
			s.Skills.ServiceSkills = make([]SkillResult, 5)
		}),
	})
}

// Reenlistment allows no DMs (p. 6), and the 12-exactly rule (pp. 6-7)
// reads the bare dice, so the data must not carry any.
func TestValidateThrows(t *testing.T) {
	rejects(t, validateThrows, map[string]*Service{
		"reenlist carrying a DM": broken(func(s *Service) {
			s.Reenlist.DMs = []DM{{Characteristic: Education, Min: 8, DM: 1}}
		}),
		"a commission target that does not parse": broken(func(s *Service) {
			s.Commission = &ThrowSpec{Target: "99+"}
		}),
	})
}

func TestValidateThrowSpec(t *testing.T) {
	rejects(t, func(spec *ThrowSpec) error { return validateThrowSpec("survival", spec) },
		map[string]*ThrowSpec{
			"a DM on an unknown characteristic": {
				Target: "5+", DMs: []DM{{Characteristic: "luck", Min: 8, DM: 1}},
			},
			"a DM of zero, which modifies nothing": {
				Target: "5+", DMs: []DM{{Characteristic: Education, Min: 8, DM: 0}},
			},
			"a threshold below 1": {
				Target: "5+", DMs: []DM{{Characteristic: Education, Min: 0, DM: 1}},
			},
			"a threshold above 15": {
				Target: "5+", DMs: []DM{{Characteristic: Education, Min: 16, DM: 1}},
			},
		})
}

// Commissions and promotions are non-existent in the Scout and Other
// services (p. 10), and a promotion without a commission path would be
// unreachable (p. 6) — so the two are present together or absent together.
func TestValidateRankStructure(t *testing.T) {
	rejects(t, validateRankStructure, map[string]*Service{
		"a commission with no promotion": broken(func(s *Service) { s.Promotion = nil }),
		"a promotion with no commission": broken(func(s *Service) { s.Commission = nil }),
		"a commission path with no ranks": broken(func(s *Service) {
			s.Ranks = nil
		}),
		"ranks with no commission path": broken(func(s *Service) {
			s.Commission, s.Promotion = nil, nil
		}),
	})
}

func TestValidateAutoSkills(t *testing.T) {
	rejects(t, validateAutoSkills, map[string]*Service{
		"a rank beyond the service's ranks": broken(func(s *Service) {
			s.AutoSkills = []AutoSkill{{Rank: 4, Skill: "Pilot"}}
		}),
		"a negative rank": broken(func(s *Service) {
			s.AutoSkills = []AutoSkill{{Rank: -1, Skill: "Pilot"}}
		}),
		"both a skill and an alteration": broken(func(s *Service) {
			s.AutoSkills = []AutoSkill{{Rank: 0, Skill: "Pilot", Characteristic: Education, Delta: 1}}
		}),
		"neither a skill nor an alteration": broken(func(s *Service) {
			s.AutoSkills = []AutoSkill{{Rank: 0}}
		}),
		"an alteration of more than +1, which p. 23 never grants": broken(func(s *Service) {
			s.AutoSkills = []AutoSkill{{Rank: 0, Characteristic: SocialStanding, Delta: 2}}
		}),
		"an alteration on an unknown characteristic": broken(func(s *Service) {
			s.AutoSkills = []AutoSkill{{Rank: 0, Characteristic: "luck", Delta: 1}}
		}),
		"a weapon category outside blade and gun": broken(func(s *Service) {
			s.AutoSkills = []AutoSkill{{Rank: 0, Skill: "Rifle", Category: "polearm"}}
		}),
		"a category on an alteration rather than a weapon": broken(func(s *Service) {
			s.AutoSkills = []AutoSkill{{Rank: 0, Characteristic: SocialStanding, Delta: 1, Category: "gun"}}
		}),
	})
}

func TestValidateMuster(t *testing.T) {
	sevenRows := func() []Benefit { return make([]Benefit, 7) }

	rejects(t, validateMuster, map[string]*Muster{
		"a benefits table that is not seven rows": {
			Benefits: make([]Benefit, 6), Cash: []int{1, 2, 3, 4, 5, 6, 7},
		},
		"a cash table that is not seven rows": {
			Benefits: sevenRows(), Cash: []int{1, 2, 3},
		},
		"a cash row of zero credits": {
			Benefits: sevenRows(), Cash: []int{1, 2, 3, 4, 5, 6, 0},
		},
		"a benefit row setting two kinds at once": {
			Benefits: []Benefit{{Passage: "low", TravellersAid: true}, {}, {}, {}, {}, {}, {}},
			Cash:     []int{1, 2, 3, 4, 5, 6, 7},
		},
	})
}

func TestValidateBenefit(t *testing.T) {
	rejects(t, validateBenefit, map[string]Benefit{
		"a passage class the book does not print":       {Passage: "steerage"},
		"a weapon category outside blade and gun":       {Weapon: "polearm"},
		"a ship that is neither kind":                   {Ship: "battleship"},
		"a delta with no characteristic to apply it to": {Delta: 2},
		"an alteration on an unknown characteristic":    {Characteristic: "luck", Delta: 1},
		"an alteration larger than p. 9 grants":         {Characteristic: Education, Delta: 3},
		"two kinds on one row":                          {Passage: "low", Ship: "scout"},
	})

	// The blank rows of Table 1 are a real result, not a defect (p. 9).
	if err := validateBenefit(Benefit{}); err != nil {
		t.Errorf("the table's blank row is a benefit of none, not an error: %v", err)
	}
}

func TestValidateSkillResult(t *testing.T) {
	rejects(t, validateSkillResult, map[string]SkillResult{
		"a row that results in nothing at all":       {},
		"a row that is both a skill and a weapon":    {Skill: "Gambling", Weapon: "blade"},
		"a weapon category outside blade and gun":    {Weapon: "polearm"},
		"an alteration on an unknown characteristic": {Characteristic: "luck", Delta: 1},
		"an alteration of more than one point":       {Characteristic: Education, Delta: 2},
		"a delta with no characteristic":             {Delta: 1},
	})

	// ±1 is what p. 11 prints, in both directions.
	for _, delta := range []int{1, -1} {
		if err := validateSkillResult(SkillResult{Characteristic: Education, Delta: delta}); err != nil {
			t.Errorf("alteration %+d rejected: %v", delta, err)
		}
	}
}

// The lookups that report what the data does not hold.
func TestLookupsReportWhatIsMissing(t *testing.T) {
	reg := registry(gambler("Navy", 1))

	if _, err := reg.Service("pirates"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Service(pirates) = %v, want ErrUnavailable", err)
	}

	if _, err := reg.ByDraftNumber(6); !errors.Is(err, ErrUnavailable) {
		t.Errorf("ByDraftNumber(6) = %v, want ErrUnavailable", err)
	}

	if _, err := reg.Weapons("polearm"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("Weapons(polearm) = %v, want ErrUnavailable", err)
	}

	if _, ok := (&SkillTables{}).Table("nonesuch"); ok {
		t.Error("Table(nonesuch) reported a table that does not exist")
	}
}

// A misspelled key must fail the build rather than zeroing a field and
// taking a rule with it.
func TestDecodeStrictRejectsUnknownFields(t *testing.T) {
	var parsed struct {
		Name string `json:"name"`
	}

	if err := decodeStrict([]byte(`{"name": "Navy", "naem": "Navy"}`), &parsed); err == nil {
		t.Fatal("want an error for an unknown field, got nil")
	}
}
