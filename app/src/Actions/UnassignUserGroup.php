<?php
namespace SvnAdmin\Actions;

class UnassignUserGroup extends Action
{
    public function execute(): void
    {
        $this->engine->forwardInvalidModule( !$this->engine->isGroupEditActive() );

        $selusers = get_request_var('selected_users');
        $selgroups = get_request_var('selected_groups');

        // Remove empty selections.
        if ($selusers != NULL && is_array($selusers))
        	$selusers = if_array_remove_empty_values($selusers);

        if ($selgroups != NULL && is_array($selgroups))
          $selgroups = if_array_remove_empty_values($selgroups);


        // Validation.
        if ($selusers == NULL || $selgroups == NULL)
        {
        	$this->engine->addException(new ValidationException(tr("You have to select at least one user and one group.")));
        }
        else
        {
        	try {
        	  // Count of: All, Done, Failed
        	  $cntAll = count($selgroups) * count($selusers);

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

        	      if( $this->engine->getGroupEditProvider()->removeUserFromGroup( $oU, $oG ) )
        	      {
        	      	$this->engine->addMessage(tr("Removed user %0 from group %1", array($oU->name, $oG->name)));
        	      }
        	      else
        	      {
        	      	$this->engine->addException(tr("Could not remove user %0 from group %1", array($oU->name, $oG->name)));
        	      }
        	    } //for
        	  } //for
        	  $this->engine->getGroupEditProvider()->save();
        	}
        	catch (Exception $ex) {
        		$this->engine->addException($ex);
        	}
        }
    }
}
