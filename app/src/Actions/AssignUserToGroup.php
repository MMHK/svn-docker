<?php
namespace SvnAdmin\Actions;

class AssignUserToGroup extends Action
{
    public function execute(): void
    {
        $this->engine->forwardInvalidModule( !$this->engine->isGroupEditActive() );

        // Parameters.
        $selusers = get_request_var('selusers');
        $selgroups = get_request_var('selgroups');

        if ($selusers == NULL)
        	$selusers = get_request_var("selected_users");

        if ($selgroups == NULL)
          $selgroups = get_request_var("selected_groups");

        if ($selusers != NULL && count($selusers) > 0 && empty($selusers[0]))
        {
        	$selusers = NULL;
        }

        if ($selgroups != NULL && count($selgroups) > 0 && empty($selgroups[0]))
        {
        	$selgroups = NULL;
        }

        // Validate.
        if( $selusers == NULL || $selgroups == NULL )
        {
          $this->engine->addException(new ValidationException(tr("You have to select at least one user and one group.")));
        }
        // Do assignments.
        else
        {
          // Count of: All, Done, Failed
          $cntAll = count($selgroups) * count($selusers);
          try
          {
            // Iterate all selected users and groups.
            for( $i=0; $i<count($selgroups); $i++ )
            {
              $oG = new \SvnAdmin\Entities\Group;
              $oG->id = $selgroups[$i];
              $oG->name = $selgroups[$i];

              for( $j=0; $j<count($selusers); $j++ )
              {
                $oU = new \SvnAdmin\Entities\User;
                $oU->id = $selusers[$j];
                $oU->name = $selusers[$j];

                if ($this->engine->getGroupEditProvider()->assignUserToGroup($oU, $oG))
                {
                  $this->engine->getGroupEditProvider()->save();
                  $this->engine->addMessage(tr("The user %0 is now a member of group %1", array($oU->name, $oG->name)));
                }
                else
                {
                  $this->engine->addException(new Exception(tr("Can not add user %0 as member of group %1.", array($oU->name, $oG->name))));
                }
              } //for
            } //for
          }
          catch (Exception $ex)
          {
            $this->engine->addException($ex);
          }
        }
    }
}
