<?php
namespace SvnAdmin\Actions;

class DeleteGroup extends Action
{
    public function execute(): void
    {
        $this->engine->forwardInvalidModule( !$this->engine->isGroupEditActive() );


        $selected = get_request_var('selected_groups');
        if($selected == NULL)
        {
          $this->engine->addException(new ValidationException(tr("You have to select at least one group.")));
        }
        else
        {
          try
          {
            // Iterate all selected users and delete them.
            for($i=0; $i<count($selected); $i++)
            {
              $g = new \SvnAdmin\Entities\Group;
              $g->id = $selected[$i];
              $g->name = $selected[$i];

              if ($this->engine->deleteGroup($g))
              {
                $this->engine->addMessage(tr("Deleted group %0 successfully.", array($g->name)));
              }
              else
              {
                $this->engine->addException(new Exception(tr("Can not remove group %0.", array($g->name))));
              }
            }
          }
          catch (Exception $ex)
          {
            $this->engine->addException($ex);
          }
        }
    }
}
