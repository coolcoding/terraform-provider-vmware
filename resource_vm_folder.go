package main

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25"
	"golang.org/x/net/context"
	"fmt"
	"github.com/vmware/govmomi/vim25/types"
	"github.com/vmware/govmomi/object"
	"strings"
	"github.com/vmware/govmomi/vim25/mo"
)

func resourceVmFolder() *schema.Resource {
	return &schema.Resource{
		Create: resourceVmFolderCreate,
		Read:   resourceVmFolderRead,
		Update: resourceVmFolderUpdate,
		Delete: resourceVmFolderDelete,

		Schema: map[string]*schema.Schema{
			// TODO: move to provider parameters
			"datacenter": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"parent": {
				Type:     schema.TypeString,
				Required: true,
				StateFunc: func(v interface{}) string {
					value := v.(string)
					path :=  strings.Trim(value, "/")
					return strings.Join([]string{"/", path}, "")
				},
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			// TODO: create parent folders?
		},
	}
}

func resourceVmFolderCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*vim25.Client)
	finder := find.NewFinder(client, false)
	ctx := context.TODO()

	datacenter := d.Get("datacenter").(string)
	parent_name := d.Get("parent").(string)
	name := d.Get("name").(string)

	path := strings.Join([]string{datacenter, "vm", parent_name}, "/")

	parent_folder, err := finder.Folder(ctx, path)
	if err != nil {
		return fmt.Errorf("Cannot find parent folder: %s", err)
	}

	folder, err := parent_folder.CreateFolder(ctx, name)
	if err != nil {
		return fmt.Errorf("Cannot create folder: %s", err)
	}

	d.SetId(folder.Reference().Value)
	return nil
}

func resourceVmFolderRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*vim25.Client)
	finder := find.NewFinder(client, false)
	ctx := context.TODO()

	mor := types.ManagedObjectReference{Type: "Folder", Value: d.Id()}
	obj, err := finder.ObjectReference(ctx, mor)
	if err != nil {
		d.SetId("")
		return nil
	}
	folder := obj.(*object.Folder)


	var o mo.ManagedEntity
	err = folder.Properties(ctx, folder.Reference(), []string{"name", "parent"}, &o)
	if err != nil {
		return fmt.Errorf("Cannot read folder: %s", err)
	}
	d.Set("name", o.Name)

	obj, err = finder.ObjectReference(ctx, *o.Parent)
	if err != nil {
		return fmt.Errorf("Cannot read folder: %s", err)
	}
	parent_folder := obj.(*object.Folder)
	datacenter := d.Get("datacenter").(string)
	prefix := strings.Join([]string{"/", datacenter, "/vm"}, "")
	path := strings.TrimPrefix(parent_folder.InventoryPath, prefix)
	path =  strings.TrimPrefix(path, "/")
	path =  strings.Join([]string{"/", path}, "")
	d.Set("parent", path)

	return nil
}

func resourceVmFolderUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*vim25.Client)
	finder := find.NewFinder(client, false)
	ctx := context.TODO()

	mor := types.ManagedObjectReference{Type: "Folder", Value: d.Id()}
	obj, err := finder.ObjectReference(ctx, mor)
	if err != nil {
		d.SetId("")
		return nil
	}
	folder := obj.(*object.Folder)

	d.Partial(true)
	if d.HasChange("name") {
		name := d.Get("name").(string)
		task, err := folder.Rename(ctx, name)
		if err != nil {
			return fmt.Errorf("Cannot rename folder: %s", err)
		}
		_, err = task.WaitForResult(ctx, nil)
		if err != nil {
			return fmt.Errorf("Cannot rename folder: %s", err)
		}
		d.SetPartial("name")
	}

	if d.HasChange("parent") {
		datacenter := d.Get("datacenter").(string)
		parent_name := d.Get("parent").(string)

		path := strings.Join([]string{datacenter, "vm", parent_name}, "/")
		parent_folder, err := finder.Folder(ctx, path)
		if err != nil {
			return fmt.Errorf("Cannot find parent folder: %s", err)
		}

		task, err := parent_folder.MoveInto(ctx, []types.ManagedObjectReference{folder.Reference()})
		if err != nil {
			return fmt.Errorf("Cannot move folder: %s", err)
		}
		_, err = task.WaitForResult(ctx, nil)
		if err != nil {
			return fmt.Errorf("Cannot move folder: %s", err)
		}
		d.SetPartial("parent")
	}

	d.Partial(false)
	return nil
}

func resourceVmFolderDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*vim25.Client)
	finder := find.NewFinder(client, false)
	ctx := context.TODO()

	mor := types.ManagedObjectReference{Type: "Folder", Value: d.Id()}
	obj, err := finder.ObjectReference(ctx, mor)
	if err != nil {
		d.SetId("")
		return nil
	}
	folder := obj.(*object.Folder)

	if children, _ := folder.Children(ctx); len(children) > 0 {
		return fmt.Errorf("Folder is not empty")
	}

	task, err := folder.Destroy(ctx)
	if err != nil {
		return fmt.Errorf("Cannot delete folder: %s", err)
	}
	_, err = task.WaitForResult(ctx, nil)
	if err != nil {
		return fmt.Errorf("Cannot delete folder: %s", err)
	}

	d.SetId("")
	return nil
}
