package digitalocean

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func dataSourceDigitalOceanDroplets() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDigitalOceanDropletsRead,
		Schema: map[string]*schema.Schema{

			"tag": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "tag associated to the droplets",
				ValidateFunc: validation.NoZeroValues,
			},
			// computed attributes
			"droplets": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of droplet that match the tag",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "name of the droplet",
						},
						"urn": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the uniform resource name for the Droplet",
						},
						"region": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the region that the droplet instance is deployed in",
						},
						"image": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the image id or slug of the Droplet",
						},
						"size": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the current size of the Droplet",
						},
						"disk": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "the size of the droplets disk in gigabytes",
						},
						"vcpus": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "the number of virtual cpus",
						},
						"memory": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "memory of the droplet in megabytes",
						},
						"price_hourly": {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "the droplets hourly price",
						},
						"price_monthly": {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "the droplets monthly price",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "state of the droplet instance",
						},
						"locked": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "whether the droplet has been locked",
						},
						"ipv4_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the droplets public ipv4 address",
						},
						"ipv4_address_private": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the droplets private ipv4 address",
						},
						"ipv6_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the droplets public ipv6 address",
						},
						"ipv6_address_private": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "the droplets private ipv4 address",
						},
						"backups": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "whether the droplet has backups enabled",
						},
						"ipv6": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "whether the droplet has ipv6 enabled",
						},
						"private_networking": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "whether the droplet has private networking enabled",
						},
						"monitoring": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "whether the droplet has monitoring enabled",
						},
						"volume_ids": {
							Type:        schema.TypeSet,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Computed:    true,
							Description: "list of volumes attached to the droplet",
						},
						"tags": tagsDataSourceSchema(),
					},
				},
			},
		},
	}
}

func dataSourceDigitalOceanDropletsRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*CombinedConfig).godoClient()

	tag := d.Get("tag").(string)

	opts := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	dropletList := []godo.Droplet{}

	for {
		droplets, resp, err := client.Droplets.ListByTag(context.Background(), tag, opts)

		if err != nil {
			return fmt.Errorf("Error retrieving droplets: %s", err)
		}

		for _, droplet := range droplets {
			dropletList = append(dropletList, droplet)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return fmt.Errorf("Error retrieving droplets: %s", err)
		}

		opts.Page = page + 1
	}

	return dropletsDecriptionAttributes(d, dropletList, meta)
}

func dropletsDecriptionAttributes(d *schema.ResourceData, droplets []godo.Droplet, meta interface{}) error {
	var s []map[string]interface{}
	for _, droplet := range droplets {
		mapping := map[string]interface{}{
			"name":          droplet.Name,
			"urn":           droplet.URN(),
			"region":        droplet.Region.Slug,
			"size":          droplet.Size.Slug,
			"price_hourly":  droplet.Size.PriceHourly,
			"price_monthly": droplet.Size.PriceMonthly,
			"disk":          droplet.Disk,
			"vcpus":         droplet.Vcpus,
			"memory":        droplet.Memory,
			"status":        droplet.Status,
			"locked":        droplet.Locked,
		}

		if droplet.Image.Slug == "" {
			mapping["image"] = strconv.Itoa(droplet.Image.ID)
		} else {
			mapping["image"] = droplet.Image.Slug
		}

		if publicIPv4 := findIPv4AddrByType(&droplet, "public"); publicIPv4 != "" {
			mapping["ipv4_address"] = publicIPv4
		}

		if privateIPv4 := findIPv4AddrByType(&droplet, "private"); privateIPv4 != "" {
			mapping["ipv4_address_private"] = privateIPv4
		}

		if publicIPv6 := findIPv6AddrByType(&droplet, "public"); publicIPv6 != "" {
			mapping["ipv6_address"] = strings.ToLower(publicIPv6)
		}

		if privateIPv6 := findIPv6AddrByType(&droplet, "private"); privateIPv6 != "" {
			mapping["ipv6_address_private"] = strings.ToLower(privateIPv6)
		}

		if features := droplet.Features; features != nil {
			mapping["backups"] = containsDigitalOceanDropletFeature(features, "backups")
			mapping["ipv6"] = containsDigitalOceanDropletFeature(features, "ipv6")
			mapping["private_networking"] = containsDigitalOceanDropletFeature(features, "private_networking")
			mapping["monitoring"] = containsDigitalOceanDropletFeature(features, "monitoring")
		}

		mapping["volume_ids"] = flattenDigitalOceanDropletVolumeIds(droplet.VolumeIDs)
		mapping["tags"] = flattenTags(droplet.Tags)

		s = append(s, mapping)
	}

	d.SetId(resource.UniqueId())

	if err := d.Set("droplets", s); err != nil {
		return err
	}

	return nil
}
