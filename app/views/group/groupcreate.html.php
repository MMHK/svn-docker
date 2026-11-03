<?php GlobalHeader(); ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Create group"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("Create a new group to grant access to the repositories for a range of users."); ?></p>

<div class="max-w-xl">
  <form method="POST" action="groupcreate.php">

    <div class="mb-4">
      <label class="mb-1 block text-sm font-medium" for="name"><?php Translate("Group name"); ?></label>
      <input type="text" name="name" id="name" class="input">
      <p class="mt-1 text-xs text-fg-muted">
        <b><?php Translate("Valid group names are"); ?>:</b> MyReaderGroup, my_reader-group4<br>
        <b><?php Translate("Invalid group names are"); ?>:</b> My Do$$ar group, My Group, Mööne Grüüpe
      </p>
    </div>

    <div class="mb-4">
      <button type="submit" name="create" value="<?php Translate('Create'); ?>" class="btn btn-primary"><?php Translate('Create'); ?></button>
    </div>

  </form>

  <p class="text-sm">
    <a class="text-accent-fg hover:underline" href="grouplist.php">&#xAB; <?php Translate("Back to overview"); ?></a>
  </p>

</div>

<?php GlobalFooter(); ?>
