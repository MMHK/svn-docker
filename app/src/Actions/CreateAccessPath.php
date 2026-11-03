<?php
namespace SvnAdmin\Actions;

class CreateAccessPath extends Action
{
    public function execute(): void
    {
        $this->engine->forwardInvalidModule(!$this->engine->isAccessPathEditActive());
        $this->engine->checkUserAuthentication(true, ACL_MOD_ACCESSPATH, ACL_ACTION_ADD);

        // Check required fields.
        $path = get_request_var('path');
        if( $path == NULL )
        {
          $this->engine->addException(new ValidationException(tr("You have to fill out all fields.")));
        }
        else
        {
          try {
            $doCreate = true;
            $p = new \SvnAdmin\Entities\AccessPath();
            $p->path = $path;

            // Is the user restricted to some paths? (project-manager)
            if ($this->engine->isAuthenticationActive())
            {
              $currentUsername = $this->engine->getSessionUsername();
              if ($this->engine->getAclManager()->isUserAccessPathManager($currentUsername))
              {
                if (!$this->engine->getAclManager()->isUserAdminOfPath($currentUsername, $path))
                {
                  $doCreate = false;
                  $this->engine->addException(new Exception(tr("You don't have the permission to create this access path: %0", array($path))));
                }
              }
            }

            // Create now.
            if ($doCreate)
            {
              $b = $this->engine->getAccessPathEditProvider()->createAccessPath($p);
              if($b) {
                $this->engine->getAccessPathEditProvider()->save();
                $this->engine->addMessage(tr("Created AccessPath \"%0\" successfully.", array($path)));
              }
              else {
                throw new Exception(tr("An unknown error occured. Check your configuration, please."));
              }
            }
          }
          catch (Exception $ex) {
            $this->engine->addException($ex);
          }
        }
    }
}
