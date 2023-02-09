package main

type BossName string

const (
	BossNameGaruda      BossName = "Garuda"
	BossNameTitan       BossName = "Titan"
	BossNameIfrit       BossName = "Ifrit"
	BossNameLeviathan   BossName = "Leviathan"
	BossNameRamuh       BossName = "Ramuh"
	BossNameShiva       BossName = "Shiva"
	BossNameNightmare   BossName = "Nightmare"
	BossNameBismark     BossName = "Bismark"
	BossNameRavana      BossName = "Ravana"
	BossNameThordan     BossName = "Thordan"
	BossNameNidhogg     BossName = "Nidhogg"
	BossNameSephirot    BossName = "Sephirot"
	BossNameSophia      BossName = "Sophia"
	BossNameZurvan      BossName = "Zurvan"
	BossNameA4S         BossName = "A4S"
	BossNameA12S        BossName = "A12S"
	BossNameSusano      BossName = "Susano"
	BossNameSriLakshmi  BossName = "Sri Lakshmi"
	BossNameShinryu     BossName = "Shinryu"
	BossNameTsukuyomi   BossName = "Tsukuyomi"
	BossNameRathalos    BossName = "Rathalos"
	BossNameByakko      BossName = "Byakko"
	BossNameSuzaku      BossName = "Suzaku"
	BossNameSeiryu      BossName = "Seiryu"
	BossNameO4S         BossName = "O4S"
	BossNameO8S         BossName = "O8S"
	BossNameO12S        BossName = "O12S"
	BossNameTitania     BossName = "Titania"
	BossNameInnocence   BossName = "Innocence"
	BossNameHades       BossName = "Hades"
	BossNameRuby        BossName = "Ruby"
	BossNameSoS         BossName = "SoS"
	BossNameEmerald     BossName = "Emerald"
	BossNameDiamond     BossName = "Diamond"
	BossNameE4S         BossName = "E4S"
	BossNameE8S         BossName = "E8S"
	BossNameE12S        BossName = "E12S"
	BossNameZodiark     BossName = "Zodiark"
	BossNameHydaelyn    BossName = "Hydaelyn"
	BossNameEndsinger   BossName = "Endsinger"
	BossNameBarbariccia BossName = "Barbariccia"
	BossNameRubicante   BossName = "Rubicante"
	BossNameP4S         BossName = "P4S"
	BossNameP8S         BossName = "P8S"
)

type MountName string

const (
	MountNameXanthos               MountName = "Xanthos"
	MountNameGullfaxi              MountName = "Gullfaxi"
	MountNameAithon                MountName = "Aithon"
	MountNameEnbarr                MountName = "Enbarr"
	MountNameMarkab                MountName = "Markab"
	MountNameBoreas                MountName = "Boreas"
	MountNameNightmare             MountName = "Nightmare"
	MountNameWhiteLanner           MountName = "White Lanner"
	MountNameRoseLanner            MountName = "Rose Lanner"
	MountNameRoundLanner           MountName = "Round Lanner"
	MountNameDarkLanner            MountName = "Dark Lanner"
	MountNameWarringLanner         MountName = "Warring Lanner"
	MountNameSophicLanner          MountName = "Sophic Lanner"
	MountNameDemonicLanner         MountName = "Demonic Lanner"
	MountNameGobwalker             MountName = "Gobwalker"
	MountNameArrhidaeus            MountName = "Arrhidaeus"
	MountNameRevelingKamuy         MountName = "Reveling Kamuy"
	MountNameBlissfulKamuy         MountName = "Blissful Kamuy"
	MountNameLegendaryKamuy        MountName = "Legendary Kamuy"
	MountNameLunarKamuy            MountName = "Lunar Kamuy"
	MountNameRathalos              MountName = "Rathalos"
	MountNameAuspiciousKamuy       MountName = "Auspicious Kamuy"
	MountNameEuphoniousKamuy       MountName = "Euphonious Kamuy"
	MountNameHallowedKamuy         MountName = "Hallowed Kamuy"
	MountNameAlteRoite             MountName = "Alte Roite"
	MountNameAirForce              MountName = "Air Force"
	MountNameModelO                MountName = "Model O"
	MountNameFaeGwiber             MountName = "Fae Gwiber"
	MountNameInnocentGwiber        MountName = "Innocent Gwiber"
	MountNameShadowGwiber          MountName = "Shadow Gwiber"
	MountNameRubyGwiber            MountName = "Ruby Gwiber"
	MountNameGwiberOfLight         MountName = "Gwiber Of Light"
	MountNameEmeraldGwiber         MountName = "Emerald Gwiber"
	MountNameDiamondGwiber         MountName = "Diamond Gwiber"
	MountNameSkyslipper            MountName = "Skyslipper"
	MountNameRamuh                 MountName = "Ramuh"
	MountNameEden                  MountName = "Eden"
	MountNameLynxOfEternalDarkness MountName = "Lynx Of Eternal Darkness"
	MountNameLynxOfDivineLight     MountName = "Lynx Of Divine Light"
	MountNameBluefeatherLynx       MountName = "Bluefeather Lynx"
	MountNameLynxOfImperiousWind   MountName = "Lynx Of Imperious Wind"
	MountNameLynxOfRighteousFire   MountName = "Lynx Of Righteous Fire"
	MountNameDemiPhoinix           MountName = "Demi-Phoinix"
	MountNameSunforged             MountName = "Sunforged"
)

func getXivBossMountMapping() map[BossName]MountName {
	return map[BossName]MountName{
		BossNameGaruda:      MountNameXanthos,
		BossNameTitan:       MountNameGullfaxi,
		BossNameIfrit:       MountNameAithon,
		BossNameLeviathan:   MountNameEnbarr,
		BossNameRamuh:       MountNameMarkab,
		BossNameShiva:       MountNameBoreas,
		BossNameNightmare:   MountNameNightmare,
		BossNameBismark:     MountNameWhiteLanner,
		BossNameRavana:      MountNameRoseLanner,
		BossNameThordan:     MountNameRoundLanner,
		BossNameNidhogg:     MountNameDarkLanner,
		BossNameSephirot:    MountNameWarringLanner,
		BossNameSophia:      MountNameSophicLanner,
		BossNameZurvan:      MountNameDemonicLanner,
		BossNameA4S:         MountNameGobwalker,
		BossNameA12S:        MountNameArrhidaeus,
		BossNameSusano:      MountNameRevelingKamuy,
		BossNameSriLakshmi:  MountNameBlissfulKamuy,
		BossNameShinryu:     MountNameLegendaryKamuy,
		BossNameTsukuyomi:   MountNameLunarKamuy,
		BossNameRathalos:    MountNameRathalos,
		BossNameByakko:      MountNameAuspiciousKamuy,
		BossNameSuzaku:      MountNameEuphoniousKamuy,
		BossNameSeiryu:      MountNameHallowedKamuy,
		BossNameO4S:         MountNameAlteRoite,
		BossNameO8S:         MountNameAirForce,
		BossNameO12S:        MountNameModelO,
		BossNameTitania:     MountNameFaeGwiber,
		BossNameInnocence:   MountNameInnocentGwiber,
		BossNameHades:       MountNameShadowGwiber,
		BossNameRuby:        MountNameRubyGwiber,
		BossNameSoS:         MountNameGwiberOfLight,
		BossNameEmerald:     MountNameEmeraldGwiber,
		BossNameDiamond:     MountNameDiamondGwiber,
		BossNameE4S:         MountNameSkyslipper,
		BossNameE8S:         MountNameRamuh,
		BossNameE12S:        MountNameEden,
		BossNameZodiark:     MountNameLynxOfEternalDarkness,
		BossNameHydaelyn:    MountNameLynxOfDivineLight,
		BossNameEndsinger:   MountNameBluefeatherLynx,
		BossNameBarbariccia: MountNameLynxOfImperiousWind,
		BossNameRubicante:   MountNameLynxOfRighteousFire,
		BossNameP4S:         MountNameDemiPhoinix,
		BossNameP8S:         MountNameSunforged,
	}
}
