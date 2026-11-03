<?php
namespace SvnAdmin\Actions;

class CreateGroup extends Action
{
    public function execute(): void
    {
        $this->engine->forwardInvalidModule( !$this->engine->isGroupEditActive() );

        // Parameters.
        $name = get_request_var('name');

        // Validation
        if ($name == NULL)
        {
          $this->engine->addException(new ValidationException(tr("You have to fill out all fields.")));
        }
        else
        {
          // Create user object.
          $g = new \SvnAdmin\Entities\Group;
          $g->id = $name;
          $g->name = $name;

          // Create the user now.
          try
          {
            if ($this->engine->getGroupEditProvider()->addGroup($g))
            {
              $this->engine->addMessage(tr("The group %0 has been created successfully.", array($g->name)));
              $this->engine->getGroupEditProvider()->save();
            }
            else
            {
              $this->engine->addException(new Exception(tr("An unknown error occured. Check your configuration, please.")));
            }
          }
          catch (Exception $ex)
          {
            $this->engine->addException($ex);
          }
        }
    }
}
