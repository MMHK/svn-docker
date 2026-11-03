<?php GlobalHeader(); ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Group"); ?>: <?php PrintStringValue("GroupName"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("On this page you can see all assigned users to this group."); ?></p>


<?php if (IsProviderActive(PROVIDER_GROUP_VIEW) && HasAccess(ACL_MOD_GROUP, ACL_ACTION_VIEW)) : ?>
<h2 class="mb-2 mt-6 text-xl font-semibold text-fg-default"><?php Translate("Users of group"); ?></h2>
<form action="groupview.php?groupname=<?php PrintStringValue('GroupNameEncoded'); ?>" method="POST">
  <input type="hidden" name="selected_groups[]" value="<?php PrintStringValue('GroupName'); ?>">

  <?php HtmlFilterBox("groupuserlist", 1); ?>

  <div class="box mb-6 overflow-hidden">
    <div class="overflow-x-auto">
      <table id="groupuserlist" class="w-full text-sm">
        <thead class="bg-canvas-subtle text-left text-fg-muted">
          <tr>
            <th scope="col" class="w-12 px-3 py-2 font-medium">#</th>
            <th scope="col" class="px-3 py-2 font-medium"><?php Translate("User"); ?></th>
          </tr>
        </thead>

        <tbody>
        <?php foreach(GetArrayValue("UserList") as $u): ?>
        <tr class="box-row hover:bg-canvas-subtle">
          <td class="px-3 py-2">
            <?php if (IsProviderActive(PROVIDER_GROUP_EDIT) && HasAccess(ACL_MOD_GROUP, ACL_ACTION_UNASSIGN)): ?>
            <input type="checkbox" name="selected_users[]" value="<?php print($u->name); ?>" aria-label="<?php print($u->name); ?>">
            <?php endif; ?>
          </td>
          <td class="px-3 py-2">
            <a class="text-accent-fg hover:underline" href="userview.php?username=<?php print($u->getEncodedName()); ?>"><?php print($u->name); ?></a>
          </td>
        </tr>
        <?php endforeach; ?>
        </tbody>

        <tfoot>
        <tr>
          <td colspan="2" class="px-3 py-3">
            <?php if (IsProviderActive(PROVIDER_GROUP_EDIT)): ?>
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div>
                <?php if (HasAccess(ACL_MOD_GROUP, ACL_ACTION_UNASSIGN)): ?>
                <button type="submit" name="unassign" value="<?php Translate('Unassign'); ?>" class="btn"><?php Translate('Unassign'); ?></button>
                <?php endif; ?>
              </div>
              <div class="flex flex-wrap items-center justify-end gap-2">
                <?php if (HasAccess(ACL_MOD_GROUP, ACL_ACTION_ASSIGN)): ?>
                <label class="sr-only" for="selected_users"><?php Translate("User"); ?></label>
                <select name="selected_users[]" id="selected_users" class="input w-auto">
                  <option value="">--- <?php Translate("User"); ?> ---</option>
                  <?php foreach(GetArrayValue("AllUserList") as $u): ?>
                  <option value="<?php print($u->name); ?>"><?php print($u->name); ?></option>
                  <?php endforeach; ?>
                </select>
                <button type="submit" name="assign_usergroup" value="<?php Translate('Assign'); ?>" class="btn"><?php Translate('Assign'); ?></button>
                <?php endif; ?>
              </div>
            </div>
            <?php endif; ?>
          </td>
        </tr>
        </tfoot>
      </table>
    </div>
  </div>
</form>
<?php endif; ?>


<?php if (IsProviderActive(PROVIDER_ACCESSPATH_VIEW) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_VIEW)): ?>
<h2 class="mb-2 mt-6 text-xl font-semibold text-fg-default"><?php Translate("Permissions of group"); ?></h2>
<form action="groupview.php?groupname=<?php PrintStringValue('GroupNameEncoded'); ?>" method="POST">
<input type="hidden" name="selected_groups[]" value="<?php PrintStringValue('GroupNameEncoded'); ?>">

  <?php HtmlFilterBox("grouplist", 1); ?>

  <div class="box mb-6 overflow-hidden">
    <div class="overflow-x-auto">
      <table id="grouplist" class="w-full text-sm">
        <thead class="bg-canvas-subtle text-left text-fg-muted">
          <tr>
            <th scope="col" class="w-12 px-3 py-2 font-medium">#</th>
            <th scope="col" class="px-3 py-2 font-medium"><?php Translate("Access-Path"); ?></th>
            <th scope="col" class="px-3 py-2 font-medium"><?php Translate("Permission"); ?></th>
          </tr>
        </thead>

        <tbody>
        <?php foreach (GetArrayValue("AccessPathList") as $ap): ?>
        <tr class="box-row hover:bg-canvas-subtle">
          <td class="px-3 py-2">
            <?php if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_UNASSIGN)): ?>
            <input type="checkbox" name="selected_accesspaths[]" value="<?php echo $ap->getPath(); ?>" aria-label="<?php echo $ap->getPath(); ?>">
            <?php endif; ?>
          </td>
          <td class="px-3 py-2">
            <a class="text-accent-fg hover:underline" href="accesspathview.php?accesspath=<?php echo $ap->getEncodedPath(); ?>"><?php echo $ap->getPath(); ?></a>
          </td>
          <td class="px-3 py-2">
            <span class="label"><?php echo $ap->getPerm(); ?></span>
          </td>
        </tr>
        <?php endforeach; ?>
        </tbody>

        <tfoot>
        <tr>
          <td colspan="3" class="px-3 py-3">
            <?php if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_UNASSIGN)): ?>
            <button type="submit" name="unassign_permission" value="<?php Translate('Unassign'); ?>" class="btn"><?php Translate('Unassign'); ?></button>
            <?php endif; ?>
          </td>
        </tr>
        </tfoot>
      </table>
    </div>
  </div>
</form>
<?php endif; ?>

<?php GlobalFooter(); ?>
