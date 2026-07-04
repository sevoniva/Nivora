package main

import (
	"fmt"

	"github.com/sevoniva/nivora/internal/usecase/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run production-like posture checks without mutating runtime state",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(cmd, configPath)
		},
	}
	cmd.Flags().StringVar(&configPath, "file", "configs/production.example.yaml", "config file to inspect")
	cmd.AddCommand(newDoctorAreaCommand("config", "Check config safety", &configPath))
	cmd.AddCommand(newDoctorAreaCommand("security", "Check security posture", &configPath))
	cmd.AddCommand(newDoctorAreaCommand("runtime", "Check runtime posture", &configPath))
	cmd.AddCommand(newDoctorAreaCommand("database", "Check database/runtime-store posture", &configPath))
	cmd.AddCommand(newDoctorAreaCommand("runners", "Check runner posture", &configPath))
	cmd.AddCommand(newDoctorAreaCommand("audit", "Check audit posture", &configPath))
	return cmd
}

func newDoctorAreaCommand(name string, short string, parentPath *string) *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := configPath
			if path == "" && parentPath != nil {
				path = *parentPath
			}
			report, err := doctor.CheckConfigFile(path)
			if err != nil {
				return err
			}
			report.Checks = filterDoctorChecks(report.Checks, doctorArea(name))
			report.Status = recomputeDoctorStatus(report.Checks)
			printJSON(cmd.OutOrStdout(), report)
			if report.Status == doctor.StatusFail {
				return fmt.Errorf("doctor %s checks failed", name)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "file", "", "config file to inspect")
	return cmd
}

func runDoctor(cmd *cobra.Command, path string) error {
	report, err := doctor.CheckConfigFile(path)
	if err != nil {
		return err
	}
	printJSON(cmd.OutOrStdout(), report)
	if report.Status == doctor.StatusFail {
		return fmt.Errorf("doctor checks failed")
	}
	return nil
}

func doctorArea(command string) string {
	return command
}

func filterDoctorChecks(checks []doctor.Check, area string) []doctor.Check {
	out := make([]doctor.Check, 0, len(checks))
	for _, check := range checks {
		if area == "" || check.Area == area || check.ID == "config.validate" {
			out = append(out, check)
		}
	}
	return out
}

func recomputeDoctorStatus(checks []doctor.Check) string {
	status := doctor.StatusPass
	for _, check := range checks {
		if check.Status == doctor.StatusFail {
			return doctor.StatusFail
		}
		if check.Status == doctor.StatusWarn {
			status = doctor.StatusWarn
		}
	}
	return status
}
