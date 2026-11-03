<?php GlobalHeader(); ?>

<?php /* 全選改用 assets/app.js：任何 id 以 selectall 開頭的核取方塊會選取同表格內的項目 */ ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("User &lt;&gt; Group assignment"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("Here you can assign users to groups."); ?></p>

<form action="usergroupassign.php" method="POST">
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
            <?php foreach (GetArrayValue("UserList") as $u) : ?>
            <tr class="box-row hover:bg-canvas-subtle">
              <td class="px-3 py-2"><input type="checkbox" name="selusers[]" value="<?php print($u->name); ?>" aria-label="<?php print($u->name); ?>"></td>
              <td class="px-3 py-2"><a class="text-accent-fg hover:underline" href="userview.php?username=<?php print($u->getEncodedName()); ?>"><?php print($u->name); ?></a></td>
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
            <?php foreach(GetArrayValue("GroupList") as $g) : ?>
            <tr class="box-row hover:bg-canvas-subtle">
              <td class="px-3 py-2"><input type="checkbox" name="selgroups[]" value="<?php print($g->name); ?>" aria-label="<?php print($g->name); ?>"></td>
              <td class="px-3 py-2"><a class="text-accent-fg hover:underline" href="groupview.php?groupname=<?php print($g->getEncodedName()); ?>"><?php print($g->name); ?></a></td>
            </tr>
            <?php endforeach; ?>
            </tbody>
          </table>
        </div>
      </div>
    </div>

  </div>

  <?php if (IsProviderActive(PROVIDER_GROUP_EDIT)) : ?>
  <div class="mt-4">
    <button type="submit" name="assign" value="<?php Translate('Assign'); ?>" class="btn btn-primary"><?php Translate('Assign'); ?></button>
  </div>
  <?php endif; ?>

</form>

<p class="mt-4 text-sm">
  <a class="text-accent-fg hover:underline" href="grouplist.php">&#xAB; <?php Translate("Back to overview"); ?></a>
</p>

<?php GlobalFooter(); ?>
