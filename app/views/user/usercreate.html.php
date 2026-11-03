<?php GlobalHeader(); ?>

<h1 class="mb-1 text-2xl font-semibold text-fg-default"><?php Translate("Create user"); ?></h1>
<p class="mb-4 text-sm text-fg-muted"><?php Translate("Create a new user to grant access to the repositories."); ?></p>

<div class="max-w-xl">
  <form method="POST" action="usercreate.php">

    <div class="mb-4">
      <label class="mb-1 block text-sm font-medium" for="username"><?php Translate("Username"); ?></label>
      <input type="text" name="username" id="username" class="input" autocomplete="username">
    </div>

    <div class="mb-4">
      <label class="mb-1 block text-sm font-medium" for="password"><?php Translate("Password"); ?></label>
      <input type="password" name="password" id="password" class="input" autocomplete="new-password">
    </div>

    <div class="mb-4">
      <label class="mb-1 block text-sm font-medium" for="password2"><?php Translate("Re-type password"); ?></label>
      <input type="password" name="password2" id="password2" class="input" autocomplete="new-password">
    </div>

    <div class="mb-4">
      <button type="submit" name="create" value="<?php Translate("Create"); ?>" class="btn btn-primary"><?php Translate("Create"); ?></button>
    </div>
  </form>

  <p class="text-sm">
    <a class="text-accent-fg hover:underline" href="userlist.php">&#xAB; <?php Translate("Back to overview"); ?></a>
  </p>

</div>

<?php GlobalFooter(); ?>
