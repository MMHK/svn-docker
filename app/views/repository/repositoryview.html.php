<?php GlobalHeader(); ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Repository browsing"); ?></h1>
<p class="mb-4 text-sm text-fg-muted">
  <a class="text-accent-fg hover:underline" href="repositoryview.php?pi=<?php print(GetValue("Repository")->getEncodedParentIdentifier()); ?>&amp;r=<?php print(GetValue("Repository")->getEncodedName()); ?>"><?php print(GetValue("Repository")->getName()); ?></a>
  : <code class="rounded bg-canvas-subtle px-1 py-0.5 text-xs"><?php PrintStringValue("CurrentPath"); ?></code>
</p>

<div class="box mb-6 overflow-hidden">
<div class="overflow-x-auto">
<table class="w-full text-sm">
<thead class="bg-canvas-subtle text-left text-fg-muted">
  <tr>
    <th scope="col" class="w-12 px-3 py-2 font-medium">
      <?php if (GetBoolValue("RepositoryRoot")) { ?>
      <a href="repositorylist.php"><img src="templates/icons/back.png" alt="TR{Back}}"></a>
      <?php } else { ?>
      <a href="repositoryview.php?pi=<?php print(GetValue("Repository")->getEncodedParentIdentifier()); ?>&amp;r=<?php print(GetValue("Repository")->getEncodedName()); ?>&amp;p=<?php PrintStringValue("BackLinkPathEncoded"); ?>"><img src="templates/icons/back.png" alt="<?php Translate("Back"); ?>"></a>
      <?php } ?>
    </th>
    <th scope="col" class="w-12 px-3 py-2 font-medium">#</th>
    <th scope="col" class="px-3 py-2 font-medium"><?php Translate("File name"); ?></th>
    <th scope="col" class="w-40 px-3 py-2 font-medium"><?php Translate("Author"); ?></th>
    <th scope="col" class="w-24 px-3 py-2 font-medium"><?php Translate("Revision"); ?></th>
    <?php if (GetBoolValue("ApacheWebLink") || GetBoolValue("CustomWebLink")) { ?>
    <th scope="col" class="w-24 px-3 py-2 font-medium"><?php Translate("Options"); ?></th>
    <?php } ?>
  </tr>
</thead>

<tbody>
  <?php foreach (GetArrayValue("ItemList") as $item) { ?>
  <tr class="box-row hover:bg-canvas-subtle">
    <td class="px-3 py-2">
      <?php if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_ADD)) { ?>
	    <a href="accesspathcreate.php?pi=<?php print(GetValue("Repository")->getEncodedParentIdentifier()); ?>&amp;r=<?php print(GetValue("Repository")->getEncodedName()); ?>&amp;p=<?php print($item->getEncodedRelativePath()); ?>"><img src="templates/icons/addpath.png" alt="-" title="<?php Translate("Add access path"); ?>"></a>
      <?php } ?>
    </td>
    <td class="px-3 py-2">
      <?php if ($item->type == 0) { ?>
      <img src="templates/icons/folder.png" border="0" alt="-">
      <?php } else { ?>
      <img src="templates/icons/file.png" border="0" alt="-">
      <?php } ?>
    </td>
    <td class="px-3 py-2">
    	<?php if ($item->type == 0) { ?>
      <a class="text-accent-fg hover:underline" href="repositoryview.php?pi=<?php print(GetValue("Repository")->getEncodedParentIdentifier()); ?>&amp;r=<?php print(GetValue("Repository")->getEncodedName()); ?>&amp;p=<?php print($item->getEncodedRelativePath()); ?>"><?php print($item->name); ?></a>
      <?php } else { ?>
			<?php print($item->name); ?>
      <?php } ?>
    </td>
    <td class="px-3 py-2 text-fg-muted">
    	<?php print($item->author); ?>
    </td>
    <td class="px-3 py-2 text-right tabular-nums">
    	<?php print($item->revision); ?>
    </td>
    <?php if (GetBoolValue("ApacheWebLink") || GetBoolValue("CustomWebLink")) { ?>
    <td class="px-3 py-2 text-center">
      <?php if (GetBoolValue("ApacheWebLink")) { ?><a href="<?php print($item->apacheWebLink); ?>" target="_blank"><img src="templates/images/apache-icon.png" alt="A" title="Apache WebDAV"></a><?php } ?>
      <?php if (GetBoolValue("CustomWebLink")) { ?><a href="<?php print($item->customWebLink); ?>" target="_blank"><img src="templates/images/weblink-icon.gif" alt="W" title="Custom Subversion Browser"></a><?php } ?>
    </td>
    <?php } ?>
  </tr>
  <?php } ?>
</tbody>

<tfoot>
  <tr>
    <td class="px-3 py-2">
    	<?php if (GetBoolValue("RepositoryRoot")) { ?>
      <a href="repositorylist.php"><img src="templates/icons/back.png" alt="TR{Back}}"></a>
    	<?php } else { ?>
      <a href="repositoryview.php?pi=<?php print(GetValue("Repository")->getEncodedParentIdentifier()); ?>&amp;r=<?php print(GetValue("Repository")->getEncodedName()); ?>&amp;p=<?php PrintStringValue("BackLinkPathEncoded"); ?>"><img src="templates/icons/back.png" alt="<?php Translate("Back"); ?>"></a>
      <?php } ?>
    </td>
    <td class="px-3 py-2"></td>
    <td class="px-3 py-2"></td>
    <td class="px-3 py-2"></td>
    <td class="px-3 py-2"></td>
    <?php if (GetBoolValue("ApacheWebLink") || GetBoolValue("CustomWebLink")) { ?>
    <td class="px-3 py-2"></td>
    <?php } ?>
  </tr>
</tfoot>

</table>
</div>
</div>

<?php GlobalFooter(); ?>
