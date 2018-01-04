// Copyright (2013) Sandia Corporation.
// Under the terms of Contract DE-AC04-94AL85000 with Sandia Corporation,
// the U.S. Government retains certain rights in this software.

package main

import (
	log "minilog"
	"os"
	"path/filepath"
)

var cmdDel = &Command{
	UsageLine: "del <reservation name>",
	Short:     "delete reservation",
	Long: `
Delete an existing reservation.
	`,
}

func init() {
	// break init cycle
	cmdDel.Run = runDel
}

// Remove the specified reservation.
func runDel(cmd *Command, args []string) {
	deleteReservation(true, args)
}

// The checkUser argument specifies whether or not we should compare the current
// username to the username of the deleted reservation. It is set to 'true' when
// a reservation is deleted at the command line, and 'false' when the reservation
// is deleted because it has expired.
func deleteReservation(checkUser bool, args []string) {
	var deletedReservation Reservation

	if len(args) != 1 {
		log.Fatalln("Invalid arguments")
	}

	user, err := getUser()
	if err != nil {
		log.Fatal("can't get current user: %v\n", err)
	}

	if checkUser {
		for _, r := range Reservations {
			if r.ResName == args[0] && r.Owner != user.Username {
				log.Fatal("You are not the owner of %v", args[0])
			}
		}
	}

	// Remove the reservation
	found := false
	for _, r := range Reservations {
		if r.ResName == args[0] {
			deletedReservation = r
			delete(Reservations, r.ID)
			found = true
		}
	}
	if !found {
		log.Fatal("Couldn't find reservation %v", args[0])
	}

	// Now purge it from the schedule
	for i, _ := range Schedule {
		for j, _ := range Schedule[i].Nodes {
			if Schedule[i].Nodes[j] == deletedReservation.ID {
				Schedule[i].Nodes[j] = 0
			}
		}
	}

	// Update the reservation file
	putReservations()
	putSchedule()

	// clean up the network config
	err = networkClear(deletedReservation.Hosts)
	if err != nil {
		log.Fatal("error clearing network isolation: %v", err)
	}

	if !igorConfig.UseCobbler {
		// Delete all the PXE files in the reservation
		for _, pxename := range deletedReservation.PXENames {
			os.Remove(filepath.Join(igorConfig.TFTPRoot, "pxelinux.cfg", pxename))
		}
	} else {
		// Set all nodes in the reservation back to the default profile
		// Cobbler commands are slow, so we run them in parallel.
		done := make(chan bool)
		f := func(h string) {
			processWrapper("cobbler", "system", "edit", "--name="+h, "--profile="+igorConfig.CobblerDefaultProfile)
			done <- true
		}
		for _, host := range deletedReservation.Hosts {
			go f(host)
		}
		for _, _ = range deletedReservation.Hosts {
			<-done
		}
		// Delete the profile and distro we created for this reservation
		if deletedReservation.CobblerProfile == "" {
			processWrapper("cobbler", "profile", "remove", "--name=igor_"+deletedReservation.ResName)
			processWrapper("cobbler", "distro", "remove", "--name=igor_"+deletedReservation.ResName)
		}
	}

	// We use this to indicate if a reservation has been created or not
	// It's used with Cobbler too, even though we don't manually manage PXE files.
	os.Remove(filepath.Join(igorConfig.TFTPRoot, "pxelinux.cfg", "igor", deletedReservation.ResName))

	// Delete the now unused kernel + initrd
	fname := filepath.Join(igorConfig.TFTPRoot, "igor", deletedReservation.ResName+"-initrd")
	os.Remove(fname)
	fname = filepath.Join(igorConfig.TFTPRoot, "igor", deletedReservation.ResName+"-kernel")
	os.Remove(fname)

	emitReservationLog("DELETED", deletedReservation)
}
