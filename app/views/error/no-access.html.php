<?php GlobalHeader(); ?>

<div class="flash flash-error">
	<h1 class="mb-1 text-xl font-semibold"><?php Translate('You do not have permission to access this page.'); ?></h1>
	<p class="text-sm">
		<?php Translate('Required permissions:'); ?><br>
		<b><?php Translate('Module:'); ?></b> <?php PrintStringValue("Module"); ?><br>
		<b><?php Translate('Action:'); ?></b> <?php PrintStringValue("Action"); ?>
	</p>
</div>

<?php GlobalFooter(); ?>
