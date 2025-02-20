// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var CommonIdsDataSourceSchema = schema.Schema{
	// This description is used by the documentation generator and the language server.
	MarkdownDescription: "Ziti Intercept Config Data Source",

	Attributes: map[string]schema.Attribute{
		"filter": schema.StringAttribute{
			MarkdownDescription: "ZitiQl filter query",
			Required:            true,
		},

		"ids": schema.ListAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "An array of allowed addresses that could be forwarded.",
			Computed:            true,
		},
	},
}
