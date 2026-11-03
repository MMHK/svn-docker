<?php GlobalHeader(); ?>

<?php /* 全選改用 assets/app.js：任何 id 以 selectall 開頭的核取方塊會選取同表格內的項目 */ ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Permissions"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("Manage the user and group permissions for the different access-paths on this site."); ?></p>

<form action="permissionassign.php" method="POST">
  <div class="grid gap-4 md:grid-cols-2">

    <div>
      <?php HtmlFilterBox("userlist", 1); ?>

      <div class="box overflow-hidden">
        <div class="box-header"><?php Translate("User"); ?></div>
        <div class="overflow-x-auto">
          <table id="userlist" class="w-full text-sm">
            <thead class="bg-canvas-subtle text-left text-fg-muted">
            <tr>
              <th scope="col" class="w-12 px-3 py-2 font-medium"><input type="checkbox" id="selectallusers" aria-label="<?php Translate("Select all"); ?>"></th>
              <th scope="col" class="px-3 py-2 font-medium"><?php Translate("User"); ?></th>
            </tr>
            </thead>
            <tbody>
            <?php foreach(GetArrayValue("UserList") as $u): ?>
            <tr class="box-row hover:bg-canvas-subtle">
              <td class="px-3 py-2"><input type="checkbox" name="selected_users[]" value="<?php print($u->getName()); ?>" aria-label="<?php print($u->getName()); ?>"></td>
              <td class="px-3 py-2"><a class="text-accent-fg hover:underline" href="userview.php?username=<?php print($u->getEncodedName()); ?>"><?php print($u->getName()); ?></a></td>
            </tr>
            <?php endforeach; ?>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <div>
      <?php HtmlFilterBox("grouplist", 1); ?>

      <div class="box overflow-hidden">
        <div class="box-header"><?php Translate("Group"); ?></div>
        <div class="overflow-x-auto">
          <table id="grouplist" class="w-full text-sm">
            <thead class="bg-canvas-subtle text-left text-fg-muted">
            <tr>
              <th scope="col" class="w-12 px-3 py-2 font-medium"><input type="checkbox" id="selectallgroups" aria-label="<?php Translate("Select all"); ?>"></th>
              <th scope="col" class="px-3 py-2 font-medium"><?php Translate("Group"); ?></th>
            </tr>
            </thead>
            <tbody>
            <?php foreach(GetArrayValue("GroupList") as $g): ?>
            <tr class="box-row hover:bg-canvas-subtle">
              <td class="px-3 py-2"><input type="checkbox" name="selected_groups[]" value="<?php print($g->getName()); ?>" aria-label="<?php print($g->getName()); ?>"></td>
              <td class="px-3 py-2"><a class="text-accent-fg hover:underline" href="groupview.php?groupname=<?php print($g->getEncodedName()); ?>"><?php print($g->getName()); ?></a></td>
            </tr>
            <?php endforeach; ?>
            </tbody>
          </table>
        </div>
      </div>
    </div>

  </div>

  <h2 class="mb-2 mt-6 text-xl font-semibold text-fg-default"><?php Translate("Select the permission for the selected user(s)/group(s)"); ?></h2>
  <div class="box overflow-hidden">
    <div class="divide-y divide-border-default">
      <label class="flex items-start gap-2 px-4 py-2 text-sm">
        <input type="radio" name="permission" value="<?php PrintStringValue("PermNone"); ?>" class="mt-1" checked>
        <span><b><?php Translate("No permission"); ?>:</b> <?php Translate("The user(s)/group(s) has no rights."); ?></span>
      </label>
      <label class="flex items-start gap-2 px-4 py-2 text-sm">
        <input type="radio" name="permission" value="<?php PrintStringValue("PermRead"); ?>" class="mt-1">
        <span><b><?php Translate("Read only"); ?>:</b> <?php Translate("The user(s)/group(s) can see all files."); ?></span>
      </label>
      <label class="flex items-start gap-2 px-4 py-2 text-sm">
        <input type="radio" name="permission" value="<?php PrintStringValue("PermReadWrite"); ?>" class="mt-1">
        <span><b><?php Translate("Read &amp; Write"); ?>:</b> <?php Translate("The user(s)/group(s) can see, create, modify and delete all files."); ?></span>
      </label>
    </div>
  </div>

  <h2 class="mb-2 mt-6 text-xl font-semibold text-fg-default"><?php Translate("Access-Path"); ?></h2>

  <?php HtmlFilterBox("accesspathlist", 1); ?>

  <div class="box overflow-hidden">
    <div class="overflow-x-auto">
      <table id="accesspathlist" class="w-full text-sm">
        <thead class="bg-canvas-subtle text-left text-fg-muted">
          <tr>
            <th scope="col" class="w-12 px-3 py-2 font-medium"><input type="checkbox" id="selectallpaths" aria-label="<?php Translate("Select all"); ?>"></th>
            <th scope="col" colspan="2" class="px-3 py-2 font-medium"><?php Translate("Access-Path"); ?></th>
          </tr>
        </thead>
        <tbody>
          <?php foreach(GetArrayValue("AccessPathList") as $p): ?>
          <tr class="box-row hover:bg-canvas-subtle">
            <td class="px-3 py-2"><input type="checkbox" name="selected_accesspaths[]" value="<?php print($p->getPath()); ?>" aria-label="<?php print($p->getPath()); ?>"></td>
            <td class="px-3 py-2" colspan="2"><a class="text-accent-fg hover:underline" href="accesspathview.php?accesspath=<?php print($p->getEncodedPath()); ?>"><?php print($p->getPath()); ?></a></td>
          </tr>
          <?php endforeach; ?>
        </tbody>
      </table>
    </div>
  </div>

  <div class="mt-4">
    <button type="submit" name="assign" value="<?php Translate("Assign"); ?>" class="btn btn-primary"><?php Translate("Assign"); ?></button>
  </div>
</form>

<?php GlobalFooter(); ?>
