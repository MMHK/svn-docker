<?php GlobalHeader(); ?>

<?php /* 預設結構說明由 assets/app.js 依 select[data-toggle-targets] 切換顯示 */ ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Create repository"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("Create a new repository to manage your sources."); ?></p>
<div class="max-w-xl">
  <form method="POST" action="repositorycreate.php">

	<div class="mb-4">
		<label class="mb-1 block text-sm font-medium" for="pi"><?php Translate('Repository location'); ?></label>
		<select name="pi" id="pi" class="input">
			<?php foreach (GetArrayValue('RepositoryParentList') as $rp) : ?>
				<option value="<?php print($rp->getEncodedIdentifier()); ?>">
					<?php print($rp->path); ?>
					<?php
					if (!empty($rp->description)) {
						print(' - ');
						print($rp->description);
					}
					?>
				</option>
			<?php endforeach; ?>
		</select>
	</div>

    <div class="mb-4">
      <label class="mb-1 block text-sm font-medium" for="reponame"><?php Translate("Repository name"); ?></label>
      <input type="text" name="reponame" id="reponame" class="input">
      <p class="mt-1 text-xs text-fg-muted">
        <b><?php Translate("Valid signs for repository name are"); ?>:</b> A-Z, a-z, 0-9, <?php Translate("Underscore"); ?>(_), <?php Translate("Hyphen"); ?>(-) <i><?php Translate("No space!"); ?></i>
      </p>
    </div>

    <div class="mb-4">
      <label class="mb-1 block text-sm font-medium" for="repotype"><?php Translate("Type"); ?></label>
      <select name="repotype" id="repotype" class="input">
        <option value="fsfs" selected="selected"><?php Translate("File System (Recommended)"); ?></option>
      </select>
    </div>

    <div class="mb-4">
      <label class="flex items-center gap-2 text-sm font-medium" for="accesspathcreate">
        <input type="checkbox" name="accesspathcreate" id="accesspathcreate" value="1" checked>
        <?php Translate("Access-Path"); ?>: <?php Translate("Create"); ?>
      </label>
    </div>

    <div class="mb-4">
      <label class="mb-1 block text-sm font-medium" for="repostructuretype"><?php Translate("Pre-defined repository structure"); ?></label>
      <select name="repostructuretype" id="repostructuretype" class="input" data-toggle-targets="repostructuretype">
        <option value=""><?php Translate("No pre-defined structure"); ?></option>
        <option value="single"><?php Translate("Single project structure"); ?></option>
        <option value="multi"><?php Translate("Multi project structure"); ?></option>
      </select>
    </div>

    <div class="mb-4 rounded-md border border-border-default bg-canvas-subtle p-3 text-sm" id="repostructuretype-single" data-toggle-group="repostructuretype" data-toggle-when="single" style="display:none;">
      <p>
        <b><?php Translate("Single project structure"); ?></b><br>
        <?php Translate("Creates the folders 'trunk', 'branches' and 'tags' in the root of the repository."); ?>
      </p>
    </div>

    <div class="mb-4 rounded-md border border-border-default bg-canvas-subtle p-3 text-sm" id="repostructuretype-multi" data-toggle-group="repostructuretype" data-toggle-when="multi" style="display:none;">
      <label class="mb-1 block font-medium" for="projectname"><?php Translate("Project name"); ?></label>
      <input type="text" name="projectname" id="projectname" value="" class="input">
      <p class="mt-1 text-xs text-fg-muted">
        <b><?php Translate("Multi project structure"); ?>:</b><br>
        <?php Translate("The folders 'trunk', 'branches' and 'tags' will be created in a subfolder, which is named by the project's name."); ?>
      </p>
    </div>

    <div class="mb-4">
      <button type="submit" name="create" value="<?php Translate("Create"); ?>" class="btn btn-primary"><?php Translate("Create"); ?></button>
    </div>
  </form>

  <p class="text-sm">
    <a class="text-accent-fg hover:underline" href="repositorylist.php">&#xAB; <?php Translate("Back to overview"); ?></a>
  </p>

</div>

<?php GlobalFooter(); ?>
