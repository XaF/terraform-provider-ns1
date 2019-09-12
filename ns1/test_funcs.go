package ns1

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"

	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
)

func testAccCheckZoneSecondary(z *dns.Zone, idx int, expected *dns.ZoneSecondaryServer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		secondary := z.Primary.Secondaries[idx]
		if z.Primary.Enabled != true {
			return fmt.Errorf("Primary.Enabled: got: false want: true")
		}
		if len(secondary.NetworkIDs) != len(expected.NetworkIDs) {
			return fmt.Errorf("Secondaries[%d].NetworkIDs: got: %d want: %d", idx, len(secondary.NetworkIDs), len(expected.NetworkIDs))
		}
		for i, v := range secondary.NetworkIDs {
			if v != expected.NetworkIDs[i] {
				return fmt.Errorf("Secondaries[%d].NetworkIDs[%d]: got: %d want %d", idx, i, v, expected.NetworkIDs[i])
			}
		}
		if secondary.IP != expected.IP {
			return fmt.Errorf("Secondaries[%d].IP: got: %s want %s", idx, secondary.IP, expected.IP)
		}
		if secondary.Port != expected.Port {
			return fmt.Errorf("Secondaries[%d].IP: got: %d want %d", idx, secondary.Port, expected.Port)
		}
		if secondary.Notify != expected.Notify {
			return fmt.Errorf("Secondaries[%d].Notify: got: %t want %t", idx, secondary.Notify, expected.Notify)
		}
		return nil
	}
}
