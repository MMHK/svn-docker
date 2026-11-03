<?php
namespace SvnAdmin\Actions;

class AssignProjectManager extends Action
{
    public function execute(): void
    {
        $this->engine->forwardInvalidModule(!$this->engine->isAuthenticationActive());
        $this->engine->checkUserAuthentication(true, ACL_MOD_PROJECTMANAGER, ACL_ACTION_ASSIGN);

        $selusers = get_request_var("selected_users");
        $selpaths = get_request_var("selected_accesspaths");

        if ($selusers == null || $selpaths == null)
        {
          $this->engine->addException(new ValidationException(tr("You have to select at least one access-path and one user.")));
        }
        else
        {
          $selusersCount = count($selusers);
          $selpathsCount = count($selpaths);

          try
          {
            for ($i=0; $i<$selusersCount; $i++)
            {
              for ($j=0; $j<$selpathsCount; $j++)
              {
                // It is not possible to set an admin for root.
                if ($selpaths[$j] == "/")
                {
                  continue;
                }

                if ($this->engine->getAclManager()->assignAccessPathAdmin($selpaths[$j], $selusers[$i]))
                  $this->engine->addMessage(tr("Assigned user %0 to access-path %1 successfully.", array($selusers[$i], $selpaths[$i])));
                else
                  $this->engine->addException(new Exception(tr("Could not assign user %0 to access-path %1", array($selusers[$i], $selpaths[$i]))));
              }
            }
            $this->engine->getAclManager()->save();
          }
          catch (Exception $ex)
          {
            $this->engine->addException($ex);
          }
        }
    }
}
