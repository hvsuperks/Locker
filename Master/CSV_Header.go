package main

var HeadersRaw = map[string]map[string][]string{
	"MD10840": {
		"AF": {"uuid", "source", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt", "slot", "result", "posture",
			"contact", "pid", "calibration", "pcal", "ncal", "mechastroke", "negepa", "posepa", "negcutoff", "poscutoff", "negepamaxcurr", "fullstroke", "slope", "hysteresis",
			"hysteresis_pos", "linearity", "linearity_pos", "codecurrent", "initcurrent", "maxcurrent", "phasemargin", "phase3db", "gainmargin", "targethalldiff",
		},

		"OIS": {"uuid", "source", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt", "port", "result",
			"x_initialize", "x_hall_calibration", "x_negvt", "x_posvt", "x_ois_loopgain", "x_ois_phasemargin", "x_ois_3db_phasemargin", "x_ois_gainmargin", "x_ois_loopgain_3rd",
			"x_centering_current", "x_max_current", "x_+stroke", "x_-stroke", "x_full_stroke", "x_moving_direction", "x_hysteresis", "x_linearity", "x_hall_decenter",
			"x_decenter", "x_sensitivity", "x_crosstalk", "x_ois_shift", "x_ois_shift_limit", "x_ois_shift_hall_max", "x_ois_shift_hall_diff", "x_hall_p2p", "x_hall_std",
			"x_finish", "y_initialize", "y_hall_calibration", "y_negvt", "y_posvt", "y_ois_loopgain", "y_ois_phasemargin", "y_ois_3db_phasemargin", "y_ois_gainmargin",
			"y_ois_loopgain_3rd", "y_centering_current", "y_max_current", "y_+stroke", "y_-stroke", "y_full_stroke", "y_moving_direction", "y_hysteresis", "y_linearity",
			"y_hall_decenter", "y_decenter", "y_sensitivity", "y_crosstalk", "y_ois_shift", "y_ois_shift_limit", "y_ois_shift_hall_max", "y_ois_shift_hall_diff",
			"y_hall_p2p", "y_hall_std",
		},

		"Tilt": {"uuid", "source", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt", "slot", "result", "flag",
			"af_pid", "af_tilt", "check_flag",
		},
	},

	"MD16849A3": {
		"AF": {"uuid", "source", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt", "slot", "result",
			"posture", "contact", "pid", "calibration", "pcal", "ncal", "mechastroke", "negepa", "posepa", "negcutoff", "poscutoff", "fullstroke", "slope", "hysteresis",
			"linearity", "codecurrent", "initcurrent", "maxcurrent", "peakcurrent", "phasemargin", "frequencypm", "phase3db", "freqpm3db", "gainmargin", "frequencygm",
		},

		"OIS": {"uuid", "source", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt", "port", "result", "x_initialize",
			"x_hall_calibration", "x_max_current", "x_+stroke", "x_-stroke", "x_full_stroke", "x_moving_direction", "x_hysteresis", "x_linearity", "x_hall_decenter", "x_decenter",
			"x_sensitivity", "x_crosstalk", "x_ois_loopgain", "x_ois_phasemargin", "x_ois_gainmargin", "y_initialize", "y_hall_calibration", "y_max_current", "y_+stroke",
			"y_-stroke", "y_full_stroke", "y_moving_direction", "y_hysteresis", "y_linearity", "y_hall_decenter", "y_decenter", "y_sensitivity", "y_crosstalk", "y_ois_loopgain",
			"y_ois_phasemargin", "y_ois_gainmargin",
		},

		"Tilt": {"uuid", "source", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt", "slot", "result", "flag",
			"af_pid", "af_tilt", "check_flag",
		},
	},

	"MD16849A6": {
		"AF": {"uuid", "source", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt", "slot", "result", "posture",
			"contact", "pid", "calibration", "pcal", "ncal", "mechastroke", "mechastrokeori", "negepa", "posepa", "negcutoff", "poscutoff", "fullstroke", "slope", "hysteresis",
			"linearity", "codecurrent", "initcurrent", "maxcurrent", "peakcurrent", "phasemargin", "gainmargin",
		},

		"OIS": {"uuid", "source", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt", "port", "result", "x_initialize",
			"x_hall_calibration", "x_max_current", "x_max_peak_current", "x_+stroke", "x_-stroke", "x_full_stroke", "x_moving_direction", "x_hysteresis", "x_linearity",
			"x_hall_decenter", "x_decenter", "x_sensitivity", "x_crosstalk", "x_circle", "x_circle_2", "x_ois_loopgain", "x_ois_phasemargin", "x_ois_gainmargin",
			"y_initialize", "y_hall_calibration", "y_max_current", "y_max_peak_current", "y_+stroke", "y_-stroke", "y_full_stroke", "y_moving_direction", "y_hysteresis",
			"y_linearity", "y_hall_decenter", "y_decenter", "y_sensitivity", "y_crosstalk", "y_circle", "y_circle_2", "y_ois_loopgain", "y_ois_phasemargin", "y_ois_gainmargin",
		},

		"Tilt": {"uuid", "source", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt", "slot", "result", "flag",
			"af_pid", "af_tilt", "check_flag",
		},
	},

	"MD12338A2": {
		"AF": {"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt",
			"slot", "result", "posture", "contact", "pid", "calibration", "pcal", "ncal", "mechastroke", "negepa", "posepa", "negcutoff",
			"poscutoff", "fullstroke", "slope", "hysteresis", "hysteresis_pos", "linearity", "linearity_pos", "codecurrent", "initcurrent",
			"maxcurrent", "posturedifference", "phasemargin", "phase3db", "gainmargin", "gainmargin_2nd"},
		"Tilt": {"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt", "slot",
			"result", "flag", "af_pid", "af_tilt", "check_flag"},
		"OIS": {"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt",
			"port", "result", "x_initialize", "x_deviceid", "x_hall_calibration", "x_centering_current", "x_max_current", "x_+stroke",
			"x_-stroke", "x_full_stroke", "x_moving_direction", "x_hysteresis", "x_linearity", "x_hall_decenter", "x_decenter",
			"x_sensitivity", "x_crosstalk", "x_ois_loopgain", "x_ois_phasemargin", "x_ois_3db_phasemargin", "x_ois_gainmargin", "x_finish",
			"y_initialize", "y_deviceid", "y_hall_calibration", "y_centering_current", "y_max_current", "y_+stroke", "y_-stroke",
			"y_full_stroke", "y_moving_direction", "y_hysteresis", "y_linearity", "y_hall_decenter", "y_decenter", "y_sensitivity",
			"y_crosstalk", "y_ois_loopgain", "y_ois_phasemargin", "y_ois_3db_phasemargin", "y_ois_gainmargin", "y_finish",
		},
	},

	"MD12338A3": {
		"AF": {"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt",
			"slot", "result", "posture", "contact", "pid", "calibration", "pcal", "ncal", "mechastroke", "negepa", "posepa", "negcutoff",
			"poscutoff", "fullstroke", "slope", "hysteresis", "hysteresis_pos", "linearity", "linearity_pos", "codecurrent", "initcurrent",
			"maxcurrent", "posturedifference", "phasemargin", "phase3db", "gainmargin", "gainmargin_2nd"},
		"Tilt": {"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt", "slot",
			"result", "flag", "af_pid", "af_tilt", "check_flag"},
		"OIS": {"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt",
			"port", "result", "x_initialize", "x_deviceid", "x_hall_calibration", "x_centering_current", "x_max_current", "x_+stroke",
			"x_-stroke", "x_full_stroke", "x_moving_direction", "x_hysteresis", "x_linearity", "x_hall_decenter", "x_decenter",
			"x_sensitivity", "x_crosstalk", "x_ois_loopgain", "x_ois_phasemargin", "x_ois_3db_phasemargin", "x_ois_gainmargin", "x_finish",
			"y_initialize", "y_deviceid", "y_hall_calibration", "y_centering_current", "y_max_current", "y_+stroke", "y_-stroke",
			"y_full_stroke", "y_moving_direction", "y_hysteresis", "y_linearity", "y_hall_decenter", "y_decenter", "y_sensitivity",
			"y_crosstalk", "y_ois_loopgain", "y_ois_phasemargin", "y_ois_3db_phasemargin", "y_ois_gainmargin", "y_finish",
		},
	},

	"MD15342X7": {
		"AF": {
			"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt",
			"slot", "result", "posture", "contact", "pid", "calibration", "pcal", "ncal", "mechastroke", "negepa", "posepa", "negcutoff",
			"poscutoff", "fullstroke", "slope", "hysteresis", "linearity", "codecurrent", "initcurrent", "maxcurrent",
		},
		"Tilt": {
			"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt",
			"slot", "result", "flag", "af_pid", "af_tilt", "check_flag",
		},
		"OIS": {
			"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt",
			"port", "result", "x_pcal", "x_ncal", "y_pcal", "y_ncal", "x_deviceid", "x_hall_calibration", "x_centering_current",
			"x_max_current", "x_+stroke", "x_-stroke", "x_full_stroke", "x_moving_direction", "x_hysteresis", "x_linearity", "x_hall_decenter",
			"x_decenter", "x_sensitivity", "x_crosstalk", "x_ois_loopgain", "x_ois_phasemargin", "x_ois_3db_phasemargin", "x_ois_gainmargin",
			"x_af_phasemargin", "x_af_3db_phasemargin", "x_af_gainmargin", "y_deviceid", "y_hall_calibration", "y_centering_current",
			"y_max_current", "y_+stroke", "y_-stroke", "y_full_stroke", "y_moving_direction", "y_hysteresis", "y_linearity", "y_hall_decenter",
			"y_decenter", "y_sensitivity", "y_crosstalk", "y_ois_loopgain", "y_ois_phasemargin", "y_ois_3db_phasemargin", "y_ois_gainmargin",
			"y_af_phasemargin", "y_af_3db_phasemargin", "y_af_gainmargin",
		},
	},

	"MD15342A4": {
		"AF": {
			"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt",
			"slot", "result", "posture", "contact", "pid", "calibration", "pcal", "ncal", "mechastroke", "negepa", "posepa", "negcutoff",
			"poscutoff", "fullstroke", "slope", "hysteresis", "linearity", "codecurrent", "initcurrent", "maxcurrent",
		},
		"Tilt": {
			"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt",
			"slot", "result", "flag", "af_pid", "af_tilt", "check_flag",
		},
		"OIS": {
			"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt",
			"port", "result", "x_pcal", "x_ncal", "y_pcal", "y_ncal", "x_deviceid", "x_hall_calibration", "x_centering_current",
			"x_max_current", "x_+stroke", "x_-stroke", "x_full_stroke", "x_moving_direction", "x_hysteresis", "x_linearity", "x_hall_decenter",
			"x_decenter", "x_sensitivity", "x_crosstalk", "x_ois_loopgain", "x_ois_phasemargin", "x_ois_3db_phasemargin", "x_ois_gainmargin",
			"x_af_phasemargin", "x_af_3db_phasemargin", "x_af_gainmargin", "y_deviceid", "y_hall_calibration", "y_centering_current",
			"y_max_current", "y_+stroke", "y_-stroke", "y_full_stroke", "y_moving_direction", "y_hysteresis", "y_linearity", "y_hall_decenter",
			"y_decenter", "y_sensitivity", "y_crosstalk", "y_ois_loopgain", "y_ois_phasemargin", "y_ois_3db_phasemargin", "y_ois_gainmargin",
			"y_af_phasemargin", "y_af_3db_phasemargin", "y_af_gainmargin",
		},
	},

	"MD15342E4": {
		"AF": {
			"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt",
			"slot", "result", "posture", "contact", "pid", "calibration", "pcal", "ncal", "mechastroke", "negepa", "posepa", "negcutoff",
			"poscutoff", "fullstroke", "slope", "hysteresis", "linearity", "codecurrent", "initcurrent", "maxcurrent",
		},
		"Tilt": {
			"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt",
			"slot", "result", "flag", "af_pid", "af_tilt", "check_flag",
		},
		"OIS": {
			"uuid", "plant_cd", "opr_cd", "machine_type", "machine_cd", "model", "lot_no", "prod_dt", "serial_no", "start_dt", "end_dt",
			"port", "result", "x_pcal", "x_ncal", "y_pcal", "y_ncal", "x_deviceid", "x_hall_calibration", "x_centering_current",
			"x_max_current", "x_+stroke", "x_-stroke", "x_full_stroke", "x_moving_direction", "x_hysteresis", "x_linearity", "x_hall_decenter",
			"x_decenter", "x_sensitivity", "x_crosstalk", "x_ois_loopgain", "x_ois_phasemargin", "x_ois_3db_phasemargin", "x_ois_gainmargin",
			"x_af_phasemargin", "x_af_3db_phasemargin", "x_af_gainmargin", "y_deviceid", "y_hall_calibration", "y_centering_current",
			"y_max_current", "y_+stroke", "y_-stroke", "y_full_stroke", "y_moving_direction", "y_hysteresis", "y_linearity", "y_hall_decenter",
			"y_decenter", "y_sensitivity", "y_crosstalk", "y_ois_loopgain", "y_ois_phasemargin", "y_ois_3db_phasemargin", "y_ois_gainmargin",
			"y_af_phasemargin", "y_af_3db_phasemargin", "y_af_gainmargin",
		},
	},
}
