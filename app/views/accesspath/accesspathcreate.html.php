<?php GlobalHeader(); ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Create access-path"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("Create a new access-path to grant access to users and groups."); ?></p>

<div class="max-w-xl">
	<form method="POST" action="accesspathcreate.php">

		<div class="mb-4">
			<label class="mb-1 block text-sm font-medium" for="path"><?php Translate("Path"); ?>:</label>
			<input type="text" name="path" id="path" class="input" value="<?php PrintStringValue("DefaultAccessPath"); ?>">
			<p class="mt-1 text-xs text-fg-muted">
				<b><?php Translate("Syntax"); ?>:</b> <i>&lt;RepoName&gt;:/&lt;path&gt;</i>
				<br><?php Translate("An \"/\" as access path holds permissions over all repositories."); ?>
			</p>
		</div>

		<?php if (IsProviderActive(PROVIDER_REPOSITORY_VIEW) && HasAccess(ACL_MOD_REPO, ACL_ACTION_VIEW)) : ?>
		<p class="mb-4 flex items-center gap-2 text-sm">
			<img src="templates/icons/addpath.png" border="0" alt="<?php Translate("Browse..."); ?>">
			<a class="text-accent-fg hover:underline" href="repositorylist.php"><?php Translate("Browse..."); ?></a>
		</p>
		<?php endif; ?>

		<div class="mb-4">
			<button type="submit" name="create" value="<?php Translate("Create"); ?>" class="btn btn-primary"><?php Translate("Create"); ?></button>
		</div>

	</form>
	<p class="text-sm">
		<a class="text-accent-fg hover:underline" href="accesspathslist.php">&#xAB; <?php Translate("Back to overview"); ?></a>
	</p>
</div>

<?php GlobalFooter(); ?>
