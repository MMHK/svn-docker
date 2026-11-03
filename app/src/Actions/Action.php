<?php
namespace SvnAdmin\Actions;

abstract class Action
{
    protected \SvnAdmin\Core\Engine $engine;
    protected \SvnAdmin\I18n\Translator $translator;
    protected \SvnAdmin\View\Template $template;

    public function __construct(
        \SvnAdmin\Core\Engine $engine,
        \SvnAdmin\I18n\Translator $translator,
        \SvnAdmin\View\Template $template
    ) {
        $this->engine = $engine;
        $this->translator = $translator;
        $this->template = $template;
    }

    abstract public function execute(): void;
}
