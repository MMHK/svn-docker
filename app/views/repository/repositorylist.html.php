<?php GlobalHeader(); ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Repository management"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("On this page you can view your existing repositories and create or delete an repository."); ?></p>

<?php foreach (GetArrayValue('RepositoryParentList') as $rp) : ?>

	<h2 class="mb-2 mt-6 text-xl font-semibold text-fg-default"><?php Translate('Location'); ?>: <?php print($rp->description); ?> <small class="text-sm font-normal text-fg-muted">(<?php print($rp->path); ?>)</small></h2>

	<?php HtmlFilterBox('repolist_' . $rp->identifier); ?>

	<form action="repositorylist.php" method="POST">
		<input type="hidden" name="pi" value="<?php print($rp->getEncodedIdentifier()); ?>">

		<div class="box mb-6 overflow-hidden">
			<div class="box-header"><?php Translate("Repository"); ?></div>
			<div class="overflow-x-auto">
				<table id="repolist_<?php print($rp->identifier); ?>" class="w-full text-sm">
					<thead class="bg-canvas-subtle text-left text-fg-muted">
						<tr>
							<th scope="col" class="w-12 px-3 py-2 font-medium"></th>
							<th scope="col" class="w-12 px-3 py-2 font-medium"></th>
							<th scope="col" class="px-3 py-2 font-medium">
								<?php Translate("Repository"); ?>
							</th>
						</tr>
					</thead>

					<tbody>
						<?php
							$list = GetArrayValue('RepositoryList');
							$list = $list[$rp->identifier];
							foreach ($list as $r) :
						?>
						<tr class="box-row hover:bg-canvas-subtle">
							<td class="px-3 py-2">
								<?php if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_ADD)) : ?>
									<a href="accesspathcreate.php?pi=<?php print($r->getEncodedParentIdentifier()); ?>&amp;r=<?php print($r->getEncodedName()); ?>">
										<img src="templates/icons/addpath.png" alt="<?php Translate("Add access path"); ?>" title="<?php Translate("Add access path"); ?>">
									</a>
								<?php endif; ?>
							</td>
							<td class="px-3 py-2">
								<?php if (GetBoolValue('ShowDeleteButton') && IsProviderActive(PROVIDER_REPOSITORY_EDIT) && HasAccess(ACL_MOD_REPO, ACL_ACTION_DELETE)) : ?>
									<input type="checkbox" name="selected_repos[]" value="<?php print($r->name); ?>" aria-label="<?php print($r->name); ?>">
								<?php endif; ?>
							</td>
							<td class="px-3 py-2">
								<a class="text-accent-fg hover:underline" href="repositoryview.php?pi=<?php print($r->getEncodedParentIdentifier()); ?>&amp;r=<?php print($r->getEncodedName()); ?>"><?php print($r->name); ?></a>
							</td>
						</tr>
						<?php endforeach; ?>
					</tbody>

					<?php if (GetBoolValue("ShowDeleteButton") && IsProviderActive(PROVIDER_REPOSITORY_EDIT) && HasAccess(ACL_MOD_REPO, ACL_ACTION_DELETE)): ?>
					<tfoot>
						<tr>
							<td colspan="3" class="px-3 py-3">
								<div class="flex flex-wrap items-center gap-3">
									<button type="submit" name="delete" value="<?php Translate("Delete"); ?>" class="btn btn-danger" onclick="return deletionPrompt('<?php Translate("Are you sure?"); ?>');"><?php Translate("Delete"); ?></button>

									<?php if (IsProviderActive(PROVIDER_ACCESSPATH_EDIT) && HasAccess(ACL_MOD_ACCESSPATH, ACL_ACTION_DELETE)): ?>
									<label class="flex items-center gap-1 text-xs">
										<input type="checkbox" id="delete_ap" name="delete_ap" value="1" checked> <?php Translate('+Remove configured Access-Paths'); ?>
									</label>
									<?php endif; ?>
								</div>
							</td>
						</tr>
					</tfoot>
					<?php endif; ?>

				</table>
			</div>
		</div>
	</form>

<?php endforeach; ?>

<?php GlobalFooter(); ?>
